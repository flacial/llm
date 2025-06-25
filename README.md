# llm

**llm** provides a simple way to talk to LLMs available via OpenRouter right from many programmers home (_the terminal_).

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

Set a default model or other options in `~/.llmrc.yaml` so you don't have to specify them every time.

**Setup:** Add a default model to your `~/.llmrc.yaml` file.

```yaml
# ~/.llmrc.yaml
model: openai/gpt-3.5-turbo # This will be used if -m is not specified
```

**Usage:** Now you can run `llm` without the `-m` flag.

```bash
llm "What is the capital of Sudan?"
```

### API Keys

Ensure your `LLM_API_KEY` environment variable is set, or include `api_key: "YOUR_KEY_HERE"` in your `~/.llmrc.yaml`.

```bash
# Example of setting an API key via environment variable (for current session)
export LLM_API_KEY="sk-or-..."
llm "Hello!"
```

### Verbose Mode (`-v` or `--verbose`)

See detailed output, including API requests and responses, useful for debugging.

```bash
llm -v "hello world" # For debugging use --debug
```

## Coming Soon

- Local model support
- Clearer error messages
- More tests
- Easier to use models listing/filtering

## Note

This is a personal tool. It works well, but isn't built for production workloads. Use at your own risk.
