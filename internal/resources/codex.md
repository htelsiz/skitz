# Codex CLI

`codex` start interactive session ^run
`codex "{{prompt}}"` run with prompt ^run:prompt
`codex --help` show help and options ^run

## Non-Interactive Mode

`codex -p "{{prompt}}"` print mode (non-interactive) ^run:prompt
`codex --print "{{prompt}}"` non-interactive output ^run:prompt
`codex --output-format json "{{prompt}}"` JSON output for scripts ^run:prompt
`codex --output-format stream-json "{{prompt}}"` streaming JSON ^run:prompt

## Permission Modes

`codex --permission-mode default "{{prompt}}"` default permissions ^run:prompt
`codex --permission-mode full-auto "{{prompt}}"` full autonomous mode ^run:prompt
`codex --permission-mode plan "{{prompt}}"` plan before executing ^run:prompt

## Model Selection

`codex --model gpt-4.1 "{{prompt}}"` use GPT-4.1 ^run:prompt
`codex --model o3 "{{prompt}}"` use o3 model ^run:prompt
`codex -m o3` shorthand model selection ^run

## Agent Control

`codex --max-turns 10 "{{prompt}}"` limit agentic turns ^run:prompt
`codex --allowedTools "Bash,Read,Write" "{{prompt}}"` allow only specific tools ^run:prompt
`codex --disallowedTools "Bash" "{{prompt}}"` disallow specific tools ^run:prompt

## Context & Files

`codex --read {{file}} "{{prompt}}"` include file in context ^run:file,prompt
`codex -r {{file}} "{{prompt}}"` shorthand read file ^run:file,prompt
`codex "fix bug in {{file}}"` reference file in prompt ^run:file

## Conversation

`codex --continue` resume last conversation ^run
`codex --resume {{id}}` resume specific conversation ^run:id

## Configuration

`codex config` show current configuration ^run
`codex config set {{key}} {{value}}` set config value ^run:key,value
`codex config path` show config file location ^run

## Piping & Scripting

`echo "{{prompt}}" | codex` pipe prompt via stdin ^run:prompt
`cat {{file}} | codex "summarize this"` pipe file content ^run:file
`codex --verbose "{{prompt}}"` verbose output ^run:prompt

## Common Workflows

`codex "add tests for {{file}}"` generate tests ^run:file
`codex "refactor {{func}} to be async"` refactor code ^run:func
`codex "fix the failing tests"` debug test failures ^run
`codex "explain this codebase"` understand project ^run
