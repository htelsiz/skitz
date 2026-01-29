# E2B - Code Interpreting & Sandboxing for AI Agents

E2B is an infrastructure that allows you to run AI-generated code in secure, sandboxed cloud environments.

## Core Concepts

- **Sandboxes**: Isolated cloud environments where code is executed.
- **Templates**: Blueprints for sandboxes (Dockerfile-based).

## Common Workflows

### Managing Templates
1. Initialize a template: `e2b template init`
2. Modify the generated `e2b.toml` and `Dockerfile`.
3. Build the template: `e2b template build`

### Managing Sandboxes
- List running sandboxes: `e2b sandbox list`
- Debug a sandbox: `e2b sandbox connect <sandboxID>`

## Documentation
Visit [e2b.dev/docs](https://e2b.dev/docs) for full documentation.
