package cmd

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/glamour"
	"github.com/flacial/llm/internal/llm"
	"github.com/flacial/llm/internal/log"
	"github.com/flacial/llm/internal/templating"
	"github.com/flacial/llm/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v3"
)

//go:embed default-templates/*
var defaultTemplates embed.FS

// Flags
var cfgFile string
var modelFlag string
var apiKeyFlag string
var copyToClipboardFlag bool
var promptFileFlag string
var streamingModeFlag bool
var formatOutputFlag bool
var verboseFlag bool
var logFileFlag string
var debugMode bool
var templateFlag string

// It's a global variable to allow easy mocking in tests by direct assignment
var httpClient llm.HTTPClient = &http.Client{
	Timeout: llm.DefaultTimeout,
}

var rootCmd = &cobra.Command{
	Use:   "llm [prompt] [flag]",
	Short: "Text, file, and work with LLMs from your terminal!",
	Long:  `llm is a CLI tool that allow you to chat with any LLM model on OpenRouter right from your sweet home (spoiler alert: the terminal)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Logger.Info().Msg("Starting llm")

		ctx, cancel := context.WithCancel(context.Background())
		// Cancel all goroutines, on going response consuming, and so on after function exits
		defer cancel()

		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-signalChannel
			log.Logger.Info().Msg("Stream interrupted. Cancelling...")
			cancel()
		}()

		finalPrompt, err := getPromptContent(args, promptFileFlag)
		if err != nil {
			log.Logger.Error().Err(err).Msg("Failed to get prompt content")
			return err
		}

		requestedModel := viper.GetString("model")
		aliases := viper.GetStringMapString("models.aliases")

		resolvedModel := requestedModel
		if aliasToFull, found := aliases[requestedModel]; found {
			resolvedModel = aliasToFull

			if viper.GetBool("verbose") {
				log.Logger.Info().Str("alias", requestedModel).Str("model", resolvedModel).Msg("Using alias for model.")
			}
		} else {
			log.Logger.Info().Str("model", resolvedModel).Msg("Using model.")
		}

		apiKey := viper.GetString("api_key")
		if apiKey == "" {
			log.Logger.Fatal().Msg("API key not set. Please provide it via --api-key, environment variable (OPENROUTER_API_KEY), or in ~/.llmrc.yaml") // Fatal if we want to exit immediately
			return errors.New("api key not set")
		}

		llmClient := llm.NewLLMClient(apiKey, httpClient, "")

		// 1. Read the template file from templateFlag variable
		// 2. Use text/template to fill the user_prompt_template
		// 3. Append the system_prmopt to the llm completion request messages
		// 4. Append the user message to the llm completion request
		// 5. (Optional) Add the model and temperature of the completion

		var completionMessages []llm.ChatCompletionMessage
		var finalResolvedModel = resolvedModel
		var finalTemperature *float64

		if templateFlag != "" {
			templateFilePath, err := getTemplateDirPath()
			if err != nil {
				log.Logger.Error().Err(err).Msg("Error getting template directory path")
				return err
			}

			templateFilePathFinal := filepath.Join(templateFilePath, templateFlag+".tmpl.yaml")
			templateFileBytes, err := os.ReadFile(templateFilePathFinal)
			if err != nil {
				log.Logger.Fatal().Err(err).Str("template_path", templateFilePath).Msg("Error reading template file.")
				return err
			}

			var selectedTemplate templating.Template

			err = yaml.Unmarshal(templateFileBytes, &selectedTemplate)
			if err != nil {
				log.Logger.Fatal().Err(err).Str("template_path", templateFilePath).Msg("Error unmarshalling template file. Check YAML syntax.")
				return err
			}

			if selectedTemplate.SystemMessage != "" {
				completionMessages = append(completionMessages, llm.ChatCompletionMessage{
					Role:    "system",
					Content: selectedTemplate.SystemMessage,
				})
			}

			processedUserPrompt, err := selectedTemplate.ProcessUserPromptTemplate(finalPrompt)
			if err != nil {
				log.Logger.Fatal().Err(err).Str("template_path", templateFilePath).Msg("Error processing user prompt template.")
				return err
			}

			completionMessages = append(completionMessages, llm.ChatCompletionMessage{
				Role:    "user",
				Content: processedUserPrompt,
			})
			log.Logger.Debug().Msg("Appended processed user prompt from template.")

			if selectedTemplate.Model != "" {
				finalResolvedModel = selectedTemplate.Model
				log.Logger.Debug().Str("model", finalResolvedModel).Msg("Overriding model from template.")
			}

			if selectedTemplate.Temperature != nil {
				finalTemperature = selectedTemplate.Temperature
				log.Logger.Debug().Float64("temperature", *finalTemperature).Msg("Overriding temperature from template.")
			}

		} else {
			completionMessages = append(completionMessages, llm.ChatCompletionMessage{
				Role:    "user",
				Content: finalPrompt,
			})
			log.Logger.Debug().Msg("No template used. Using direct user prompt.")
		}

		completionBody := llm.ChatCompletionRequest{
			Model:       finalResolvedModel,
			Messages:    completionMessages,
			Temperature: finalTemperature,
		}

		if !streamingModeFlag || viper.GetBool("always_format") {
			completion, err := llmClient.GetChatCompletion(ctx, completionBody)
			if err != nil {
				log.Logger.Error().Err(err).Msg("Error getting chat completion")
				return err
			}

			if len(completion.Choices) > 0 {
				completionContent := completion.Choices[0].Message.Content

				// TODO: Allow configuring the code theme/stylesheet
				// Give the output a glammm 💅
				renderedOutput, renderErr := glamour.Render(completionContent, "auto")
				if renderErr != nil {
					log.Logger.Error().Err(renderErr).Msg("Error rendering output.")
				} else {
					fmt.Println(renderedOutput)
				}

				if viper.GetBool("always_copy") {
					log.Logger.Info().Msg("Copying to clipboard...")
					err := utils.CopyToClipboard(completion.Choices[0].Message.Content)
					if err != nil {
						log.Logger.Warn().Err(err).Msg("Error copying to clipboard")
					}
				}
			} else {
				log.Logger.Warn().Msg("OpenRouter responded with no choices!")
				return errors.New("no completion choices received")
			}
		} else {
			completionBody.Stream = true
			fullCompletion, err := llmClient.GetStreamingChatCompletion(ctx, completionBody, os.Stdout)
			if err != nil {
				log.Logger.Error().Err(err).Msg("Error getting streaming chat completion")
				return err
			}

			if viper.GetBool("always_copy") {
				log.Logger.Info().Msg("Copying to clipboard...")
				err := utils.CopyToClipboard(fullCompletion)
				if err != nil {
					log.Logger.Warn().Err(err).Msg("Error copying to clipboard")
				}
			}
		}

		return nil
	},
	Args: func(cmd *cobra.Command, args []string) error {
		return nil // Allow arbitrary arguments
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initDefaultTemplates)
	cobra.OnInitialize(func() {
		log.InitLggger(viper.GetBool("verbose"), viper.GetBool("debug_mode"), viper.GetString("log_file"))
	})

	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output for debugging information.")
	rootCmd.PersistentFlags().StringVar(&logFileFlag, "log-file", "", "Path to the log file (default: ~/.llm/logs/llm.log).")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging level (overrides --verbose).")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log_file", rootCmd.PersistentFlags().Lookup("log-file"))
	viper.BindPFlag("debug_mode", rootCmd.PersistentFlags().Lookup("debug"))

	// Store the config file in a variable if provided through a flag
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.llm.yaml)")
	viper.BindPFlag("config_file", rootCmd.PersistentFlags().Lookup("config"))

	// Store "model" flag in a variable
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Specify the LLM model to use (e.g., google/gemini-2.5-flash, fast, gf25)")
	// Looking for "model" value from the flag first, then env variables, then config file, ..xetc
	viper.BindPFlag("model", rootCmd.Flags().Lookup("model"))

	// Store "api-key" flag in a variable
	rootCmd.PersistentFlags().StringVarP(&apiKeyFlag, "api-key", "k", "", "Your LLM API key (overrides config/env)")
	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))

	rootCmd.Flags().BoolVarP(&copyToClipboardFlag, "copy", "c", false, "Copy the LLM response to the clipboard")
	viper.BindPFlag("always_copy", rootCmd.Flags().Lookup("copy"))

	rootCmd.Flags().StringVarP(&promptFileFlag, "prompt-file", "f", "", "Path to a file containing the prompt")

	rootCmd.Flags().BoolVarP(&streamingModeFlag, "stream-mode", "s", true, "Show the LLM output in blocking style")
	viper.BindPFlag("use_streaming", rootCmd.Flags().Lookup("stream-mode"))

	rootCmd.Flags().BoolVarP(&formatOutputFlag, "format", "F", false, "Format the LLM output")
	viper.BindPFlag("always_format", rootCmd.Flags().Lookup("format"))

	rootCmd.Flags().StringVarP(&templateFlag, "template", "t", "", "Specify the template to use for the prompt")
	viper.BindPFlag("template", rootCmd.Flags().Lookup("template"))
}

func initConfig() {
	var configPath string
	if cfgFile != "" {
		configPath = cfgFile
	} else {
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			home, err := os.UserHomeDir()
			cobra.CheckErr(err)
			xdgConfigHome = filepath.Join(home, ".config")
		}
		configPath = filepath.Join(xdgConfigHome, "llm", "config.yaml")
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Any variables starting with LLM_* are captured for the cli
	viper.SetEnvPrefix("LLM")

	// Auto loads config files from env variables if any matches
	viper.AutomaticEnv()

	viper.SetDefault("always_format", false)
	viper.SetDefault("use_streaming", true)
	viper.SetDefault("always_copy", false)
	viper.SetDefault("api_key", "")
	viper.SetDefault("model", "google/gemini-2.5-flash")
	viper.SetDefault("verbose", false)
	viper.SetDefault("debug_mode", false)
	viper.SetDefault("log_file", "")
	viper.SetDefault("models.aliases", map[string]string{
		"fast":  "openai/gpt-4.1-nano",
		"10x":   "anthropic/claude-sonnet-4",
		"smart": "google/gemini-2.5-pro",
		"gpt4":  "openai/gpt-4o",
	})

	if err := viper.ReadInConfig(); err == nil {
		log.Logger.Info().Str("config_file", viper.ConfigFileUsed()).Msg("Using config file.")
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Logger.Info().Str("config_path", configPath).Msg("Config file not found. Creating a new one with defaults...")

			// Attempt to write the default config file
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				log.Logger.Error().Err(err).Str("config_path", configPath).Msg("Error creating config directory.")
				return
			}

			if writeErr := viper.SafeWriteConfigAs(configPath); writeErr != nil {
				log.Logger.Error().Err(writeErr).Str("config_path", configPath).Msg("Error creating default config file.")
			} else {
				log.Logger.Info().Str("config_path", configPath).Msg("Default config file created.")
			}

			if err := viper.ReadInConfig(); err != nil {
				log.Logger.Error().Err(err).Msg("Error reading newly created config file.")
			}
		} else {
			// Some unknown system error
			log.Logger.Error().Err(err).Msg("Error reading config file.")
			os.Exit(1)
		}
	}
}

func initDefaultTemplates() {
	templateDirPath, err := getTemplateDirPath()
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to get template directory path.")
		return
	}

	entries, err := os.ReadDir(templateDirPath)
	if err != nil && !os.IsNotExist(err) {
		log.Logger.Error().Err(err).Str("path", templateDirPath).Msg("Failed to read template directory.")
		return
	}

	hasTemplates := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			hasTemplates = true
			break
		}
	}

	if hasTemplates {
		log.Logger.Debug().Str("path", templateDirPath).Msg("Default templates already exist or custom templates are present. Skipping auto-initialization.")
		// Templates already exist.
		return
	}

	log.Logger.Info().Str("path", templateDirPath).Msg("Template directory is empty or missing. Initializing with default templates...")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(templateDirPath, 0755); err != nil {
		log.Logger.Error().Err(err).Str("path", templateDirPath).Msg("Failed to create template directory for defaults.")
		return
	}

	// Copy embedded templates
	fs.WalkDir(defaultTemplates, "default-templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		relPath := strings.TrimPrefix(path, "default-templates/")
		destPath := filepath.Join(templateDirPath, relPath)

		sourceFile, err := defaultTemplates.Open(path)
		if err != nil {
			log.Logger.Error().Err(err).Str("source", path).Msg("Failed to open embedded template file.")
			// Continue even if there's an error
			return nil
		}
		defer sourceFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			log.Logger.Error().Err(err).Str("dest", destPath).Msg("Failed to create destination template file.")
			return nil
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, sourceFile); err != nil {
			log.Logger.Error().Err(err).Str("source", path).Str("dest", destPath).Msg("Failed to copy embedded template file.")
			return nil
		}

		log.Logger.Debug().Str("template", relPath).Str("dest", destPath).Msg("Copied default template.")
		return nil
	})

	log.Logger.Info().Msg("Default templates initialized successfully.")
}

func getTemplateDirPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("Failed to get user home directory for templates.")
		return "", err
	}

	return filepath.Join(homeDir, ".llm", "templates"), nil
}
