package cli

// GenerateBashCompletion returns a bash completion script for gopher.
func GenerateBashCompletion() string {
	return `_gopher() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    opts="--model --query --cwd --print --continue --resume --system-prompt --system-prompt-file --append-system-prompt --max-turns --dangerously-skip-permissions --output-format --thinking --effort --verbose --version --no-session-persistence --allowed-tools --disallowed-tools --session-id --name --prefill --debug --debug-file --bare --max-budget-usd --provider --api-url --input-format --json-schema --init --fallback-model --permission-mode --add-dir --worktree --betas --include-hook-events --help"
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
}
complete -F _gopher gopher`
}

// GenerateZshCompletion returns a zsh completion script for gopher.
func GenerateZshCompletion() string {
	return `#compdef gopher
_gopher() {
    _arguments \
        '--model[Model to use]:model:' \
        '--query[One-shot query]:query:' \
        '--cwd[Working directory]:dir:_directories' \
        {-p,--print}'[Print response and exit]' \
        {-c,--continue}'[Continue most recent conversation]' \
        {-r,--resume}'[Resume by session ID]:id:' \
        '--system-prompt[Override system prompt]:prompt:' \
        '--max-turns[Maximum turns]:turns:' \
        '--output-format[Output format]:format:(text json stream-json)' \
        '--thinking[Thinking mode]:mode:(enabled disabled)' \
        '--effort[Effort level]:level:(low medium high max)' \
        '--provider[Provider]:provider:(anthropic bedrock vertex openai)' \
        '--permission-mode[Permission mode]:mode:(auto interactive deny)' \
        '--version[Show version]' \
        '--verbose[Verbose output]' \
        '--help[Show help]'
}
_gopher "$@"`
}

// GenerateFishCompletion returns a fish completion script for gopher.
func GenerateFishCompletion() string {
	return `complete -c gopher -l model -d "Model to use"
complete -c gopher -l query -d "One-shot query"
complete -c gopher -l cwd -d "Working directory" -rF
complete -c gopher -s p -l print -d "Print response and exit"
complete -c gopher -s c -l continue -d "Continue most recent conversation"
complete -c gopher -s r -l resume -d "Resume by session ID"
complete -c gopher -l output-format -d "Output format" -rfa "text json stream-json"
complete -c gopher -l thinking -d "Thinking mode" -rfa "enabled disabled"
complete -c gopher -l effort -d "Effort level" -rfa "low medium high max"
complete -c gopher -l provider -d "Provider" -rfa "anthropic bedrock vertex openai"
complete -c gopher -l version -d "Show version"
complete -c gopher -l verbose -d "Verbose output"`
}
