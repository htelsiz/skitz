# Git

`git status` show working tree status ^run
`git diff` show unstaged changes ^run
`git diff --staged` show staged changes ^run
`git add -A` stage all changes ^run
`git add {{file}}` stage file ^run:file
`git commit -m "{{msg}}"` commit with message ^run:msg
`git push` push to remote ^run
`git pull` pull from remote ^run
`git log --oneline -20` recent commits ^run
`git log --graph --oneline --all` commit graph ^run
`git branch` list branches ^run
`git branch {{name}}` create branch ^run:name
`git checkout {{branch}}` switch branch ^run:branch
`git checkout -b {{branch}}` create and switch ^run:branch
`git merge {{branch}}` merge branch ^run:branch
`git stash` stash changes ^run
`git stash pop` pop stash ^run
`git reset HEAD {{file}}` unstage file ^run:file
`git remote -v` list remotes ^run
`git fetch --all` fetch all remotes ^run
`gh pr list` list pull requests ^run
`gh pr create` create pull request ^run
`gh pr view {{num}}` view PR ^run:num
`gh issue list` list issues ^run
