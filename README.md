<img src="https://github.com/user-attachments/assets/6eaaf692-91f9-4faa-bd0a-22513089c4f3" alt="llm-logo" style="width: 200px; height: 200px;">

# llm

**llm** provides a simple way to talk to LLMs available via OpenRouter right from many programmers home (_the terminal_).

## Table of Contents

- [Install](#install)
- [Examples](#examples)
  - [Basic Usage: Ask Anything](#basic-usage-ask-anything)
  - [Model Selection (`-m` or `--model`)](#model-selection--m-or---model)
  - [Model Listing](#model-listing)
  - [Streaming Output](#streaming-output)
  - [Clipboard Copy (`-C` or `--copy`)](#clipboard-copy--c-or---copy)
  - [Templates (`-t` or `--template`)](#templates--t-or---template)
  - [Configurable](#configurable)
  - [API Keys](#api-keys)
  - [Shell Completion](#shell-completion)
  - [Verbose Mode (`-v` or `--verbose`)](#verbose-mode--v-or---verbose)
- [Coming Soon](#coming-soon)
- [Note](#note)

## Install

Clone the repository and install the `llm` executable to your Go binary path (`(go env GOPATH)/bin`):

```bash
git clone https://github.com/flacial/llm
cd llm
go install .
```

## Examples

### Basic Usage: Ask Anything

You can send a prompt in a few ways:

1.  **As a direct argument:**

    ```bash
    llm "What's the capital of Sudan?"
    ```

2.  **Via standard input (`stdin`):**

    ```bash
    echo "Tell me a short story about a brave dragon and a sleeping cow." | llm
    ```

3.  **From a file (`-f` or `--file`):**

    ```bash
    echo "Summarize the key points of the paper Attention Is All You Need." > summary.txt
    ```

    Then, run `llm` with the file:

    ```bash
    llm -f summary.txt
    ```

### Model Selection (`-m` or `--model`)

Override your default model (if set) or specify a particular model for a single query.

```bash
llm -m google/gemini-flash-1.5 "Explain the concept of recursion in programming as if I'm a grug programmer."
```

### Model Listing

View all available LLM models from OpenRouter:

```bash
llm models
```

### Streaming Output

By default, `llm` streams responses live.

```bash
llm "Write a haiku about a bustling city at sunset."
```

### Clipboard Copy (`-C` or `--copy`)

Automatically copy the LLM's response to your system clipboard.

```bash
llm -C "What is the chemical symbol for gold?"
```

_(After running, you can paste the answer (`Au`) into any text field.)_

### Templates (`-t` or `--template`)

Use predefined prompts for common tasks.

**Setup:** Add a new template to your `~/.llm/templates/` folder.

```yaml
# ~/.llm/templates/brainstorm.tmpl.yaml
name: "brainstorm"
description: "Generates creative ideas, concepts, or solutions for a given topic."
system_message: |
  You are a creative brainstorming assistant. Your role is to generate a diverse range of ideas, concepts, or solutions based on the user's input. Think broadly, explore different angles, and provide innovative suggestions. Encourage out-of-the-box thinking.
user_prompt_template: |
  I need some brainstorming ideas for:

  {{.UserPrompt}}

  Please provide at least 5 distinct ideas or approaches.
```

**Usage:** Provide a prompt and apply the template.

```bash
llm "Vacation plans for going to paris" -t brainstorm
```

### Configurable

Set a default model or other options in your configuration file so you don't have to specify them every time.

**Configuration File Location:** The configuration file is located at `$XDG_CONFIG_HOME/llm/config.yaml` (typically `~/.config/llm/config.yaml` on most systems). If `XDG_CONFIG_HOME` is not set, it defaults to `~/.config`. You can also specify a custom location using the `--config` flag.

**Setup:** Add a default model to your configuration file:

```yaml
# ~/.config/llm/config.yaml
model: openai/gpt-3.5-turbo # This will be used if -m is not specified
api_key: "your-api-key-here"  # Optional if using environment variable
```

**Model Aliases:** You can use shorter aliases for commonly used models:

```yaml
# ~/.config/llm/config.yaml
model: fast # Uses the built-in alias for openai/gpt-4.1-nano
# Or define custom aliases:
models:
  aliases:
    my-model: "anthropic/claude-3-5-sonnet"
    quick: "google/gemini-flash-1.5"
```

**Built-in Aliases:**
- `fast`: openai/gpt-4.1-nano
- `10x`: anthropic/claude-sonnet-4
- `smart`: google/gemini-2.5-pro
- `gpt4`: openai/gpt-4o

**Usage:** Now you can run `llm` without the `-m` flag:

```bash
llm "What is the capital of Sudan?"
# Or use an alias
llm -m fast "Quick question here"
```

### API Keys

Ensure your `LLM_API_KEY` environment variable is set, or include `api_key: "YOUR_KEY_HERE"` in your `~/.llmrc.yaml`.

```bash
# Example of setting an API key via environment variable (for current session)
export LLM_API_KEY="sk-or-..."
llm "Hello!"
```

### Shell Completion

### Bash:

To load completions for the current session:

```bash
$ source <(llm completion bash)
```

To load completions for each new session, execute this once:

- **Linux:**
  ```bash
  $ llm completion bash > /etc/bash_completion.d/llm
  ```
- **macOS:**
  ```bash
  $ llm completion bash > /usr/local/etc/bash_completion.d/llm
  ```

### Zsh:

To load completions for each session, execute this once:

```bash
$ llm completion zsh > ~/.zsh/_llm
```

Restart your terminal for it to work.

### Fish:

To load completions for the current session:

```bash
$ llm completion fish | source
```

To load completions for each new session, execute this once:

```bash
$ llm completion fish > ~/.config/fish/completions/llm.fish
```

### Verbose Mode (`-v` or `--verbose`)

See detailed output, including API requests and responses, useful for debugging.

```bash
llm -v "hello world" # For debugging use --debug
```

## Note

This is a personal tool. It works well, but isn't built for production workloads. Use at your own risk.
