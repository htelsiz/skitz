# Cursor

`agent` start interactive agent session ^run
`agent "{{prompt}}"` run agent with prompt ^run:prompt
`agent --mode=plan` start in plan mode (design before coding) ^run
`agent --mode=ask` start in read-only ask mode ^run
`agent -p "{{prompt}}"` non-interactive mode, print response ^run:prompt
`agent -p "{{prompt}}" --output-format json` scripted JSON output ^run:prompt
`agent ls` list previous conversations ^run
`agent resume` resume most recent session ^run
`agent --resume "{{id}}"` resume specific conversation ^run:id
`& {{message}}` send to cloud agent (pick up on web/mobile) ^run:message
