# LLMAgent

`llmasagent` is a command-line tool and MCP server for interacting with Large Language Models (LLMs). It supports direct command-line queries, an interactive TUI chat mode, and can act as an MCP (Machine Communication Protocol) server to receive problems from other LLMs or services.

## Features

*   **Single Query Mode:** Get a quick response from an LLM for a single problem.
*   **Interactive Chat Mode:** A TUI (Text User Interface) for conversational interaction with an LLM.
*   **MCP Server Mode:** Listens for problem contexts via HTTP and responds with potential solutions.
*   **Configurable LLM Provider:** Currently supports a mock LLM and is being prepared for OpenRouter integration.

## Getting Started

### Prerequisites

*   Go (latest version recommended)

### Building

1.  Clone the repository:
    ```bash
    git clone <repository-url>
    cd llmasagent
    ```
2.  Build the application:
    ```bash
    go build -o llmasagent ./cmd/llmasagent
    ```
    This will create an executable named `llmasagent` in the project root.

### Configuration

`llmasagent` is configured via environment variables:

*   `LLMAGENT_LLM_PROVIDER_TYPE`: Specifies the LLM provider.
    *   `mock` (default): Uses a built-in mock provider that echoes input. No API key needed.
    *   `openrouter`: Uses the OpenRouter.ai API. Requires `LLMAGENT_OPENROUTER_API_KEY`.
*   `LLMAGENT_OPENROUTER_API_KEY`: Your OpenRouter API key. Only used if `LLMAGENT_LLM_PROVIDER_TYPE` is `openrouter`.
*   `LLMAGENT_OPENROUTER_MODEL`: The specific model to use from OpenRouter (e.g., `gryphe/mythomax-l2-13b`). Only used if `LLMAGENT_LLM_PROVIDER_TYPE` is `openrouter`. Defaults to `gryphe/mythomax-l2-13b`.
*   `LLMAGENT_MCP_SERVER_PORT`: The port for the MCP server to listen on. Defaults to `8080`. This is used by all modes that might involve server functionality internally or externally.

### Usage

**1. Single Problem Mode:**

   Use the `-problem` flag to pass your problem description:
   ```bash
   ./llmasagent -problem "Explain quantum computing in simple terms."
   ```

**2. Interactive Chat Mode (TUI):**

   Start the TUI using the `-chat` flag:
   ```bash
   ./llmasagent -chat
   ```
   In the TUI:
   *   Type your message and press `Enter` to send.
   *   Use `Ctrl+C` or `Esc` to exit.

**3. MCP Server Mode:**

   Start the server using the `-server` flag:
   ```bash
   ./llmasagent -server
   ```
   The server will start on the port specified by `LLMAGENT_MCP_SERVER_PORT` (default 8080).

   You can then send POST requests to the `/mcp` endpoint with a JSON body:
   ```json
   {
       "problem_context": "Describe the process of photosynthesis."
   }
   ```
   Example using `curl`:
   ```bash
   curl -X POST -H "Content-Type: application/json" -d '{"problem_context":"What is the capital of France?"}' http://localhost:8080/mcp
   ```
   The server will respond with a JSON object containing the `potential_solution` or an `error`.

## Development

To ensure all dependencies are correctly managed, run:
```bash
go mod tidy
```
This is especially useful after pulling new changes or adding dependencies.