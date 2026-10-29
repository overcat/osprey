package main

import (
	"fmt"
	"io"
	"strings"
)

// names are the commands Osprey installs as.
var names = []string{"osprey", "osp"}

// completionGuide says how to install the completion scripts by hand.
const completionGuide = `osp install-cli installs them where your shell finds them. By hand:
  fish    osp completion fish > ~/.config/fish/completions/osp.fish
  zsh     osp completion zsh > _osprey, in a folder of your fpath
  bash    osp completion bash > ~/.local/share/bash-completion/completions/osp`

// refs lists the branches and tags to complete, from the shell.
const refs = `git for-each-ref --format='%(refname:short)' refs/heads refs/remotes refs/tags 2>/dev/null`

// completion writes the completion script of a shell for the commands
// named, osprey and osp when none.
func completion(w io.Writer, shell string, commands ...string) error {
	if len(commands) == 0 {
		commands = names
	}
	switch shell {
	case "fish":
		for _, name := range commands {
			c := "complete -c " + name
			fmt.Fprintf(w, "%s -n 'not __fish_seen_subcommand_from install-cli completion' -a \"(%s)\" -d ref\n", c, refs)
			fmt.Fprintf(w, "%s -s C -r -a '(__fish_complete_directories)' -d 'run as if started in this directory'\n", c)
			fmt.Fprintf(w, "%s -l committed -d 'leave out what is not committed yet'\n", c)
			fmt.Fprintf(w, "%s -s w -l wait -d 'return only when the window closes'\n", c)
			fmt.Fprintf(w, "%s -s h -l help -d 'show the help'\n", c)
			fmt.Fprintf(w, "%s -s v -l version -d 'show the version'\n", c)
			fmt.Fprintf(w, "%s -n __fish_use_subcommand -a install-cli -d 'put osprey and osp on your PATH'\n", c)
			fmt.Fprintf(w, "%s -n __fish_use_subcommand -a completion -d 'print a completion script'\n", c)
			fmt.Fprintf(w, "%s -n '__fish_seen_subcommand_from install-cli' -f -a '(__fish_complete_directories)'\n", c)
			fmt.Fprintf(w, "%s -n '__fish_seen_subcommand_from completion' -f -a 'fish zsh bash'\n", c)
		}
	case "zsh":
		// It completes when zsh loads it from a folder of fpath, and
		// registers itself when sourced.
		fmt.Fprintf(w, `#compdef %[1]s

_osprey() {
  local -a refs
  refs=(${(f)"$(%[2]s)"})
  _arguments -s \
    '-C[run as if started in this directory]:directory:_files -/' \
    '--committed[leave out what is not committed yet]' \
    '(-w --wait)'{-w,--wait}'[return only when the window closes]' \
    '(- *)'{-h,--help}'[show the help]' \
    '(- *)'{-v,--version}'[show the version]' \
    '*:branch, commit or path:{_describe -t refs ref refs; _files}'
}

if [ "$funcstack[1]" = "_osprey" ]; then
  _osprey "$@"
else
  compdef _osprey %[1]s
fi
`, strings.Join(commands, " "), refs)
	case "bash":
		fmt.Fprintf(w, `_osprey() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  if [ "$COMP_CWORD" -eq 2 ] && [ "${COMP_WORDS[1]}" = completion ]; then
    COMPREPLY=($(compgen -W "fish zsh bash" -- "$cur"))
    return
  fi
  COMPREPLY=($(compgen -W "$(%s) -C --committed -w --wait -h --help -v --version install-cli completion" -- "$cur") $(compgen -f -- "$cur"))
}
complete -F _osprey %s
`, refs, strings.Join(commands, " "))
	default:
		return fmt.Errorf("unknown shell %q: fish, zsh or bash", shell)
	}
	return nil
}
