package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long: `To load completions:

Bash:

  $ source <(llm completion bash)

  # To load completions for each session, execute once:
  # Linux:
  #   $ llm completion bash > /etc/bash_completion.d/llm
  # macOS:
  #   $ llm completion bash > /usr/local/etc/bash_completion.d/llm

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:

  #   $ echo "autoload -Uz compinit" >> ~/.zshrc
  #   $ echo "compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  #   $ llm completion zsh > ~/.zsh/_llm

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ llm completion fish | source

  # To load completions for each session, execute once:
  #   $ llm completion fish > ~/.config/fish/completions/llm.fish

PowerShell:

  Idk, I don't use wimdows
`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	completionCmd.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Generate the bash completion script",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenBashCompletion(os.Stdout)
		},
	})

	completionCmd.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Generate the zsh completion script",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenZshCompletion(os.Stdout)
		},
	})

	completionCmd.AddCommand(&cobra.Command{
		Use:   "fish",
		Short: "Generate the fish completion script",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenFishCompletion(os.Stdout, true)
		},
	})

	completionCmd.AddCommand(&cobra.Command{
		Use:   "powershell",
		Short: "Generate the powershell completion script",
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenPowerShellCompletion(os.Stdout)
		},
	})
}
