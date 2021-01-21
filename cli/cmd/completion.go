/*
Copyright 2021 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/spf13/cobra"
)

func completionInit() {
	_completionCmd.Flags().SortFlags = false
}

var _bashAliasText = `
# alias

alias cx='cortex'
if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_cortex cx
else
    complete -o default -o nospace -F __start_cortex cx
fi
`

var _completionCmd = &cobra.Command{
	Use:   "completion SHELL",
	Short: "generate shell completion scripts",
	Long: `generate shell completion scripts

to enable cortex shell completion:
    bash:
        add this to ~/.bash_profile (mac) or ~/.bashrc (linux):
            source <(cortex completion bash)

        note: bash-completion must be installed on your system; example installation instructions:
            mac:
                1) install bash completion:
                   brew install bash-completion
                2) add this to your ~/.bash_profile:
                   source $(brew --prefix)/etc/bash_completion
                3) log out and back in, or close your terminal window and reopen it
            ubuntu:
                1) install bash completion:
                   apt update && apt install -y bash-completion  # you may need sudo
                2) open ~/.bashrc and uncomment the bash completion section, or add this:
                   if [ -f /etc/bash_completion ] && ! shopt -oq posix; then . /etc/bash_completion; fi
                3) log out and back in, or close your terminal window and reopen it

    zsh:
        option 1:
            add this to ~/.zshrc:
                source <(cortex completion zsh)
            if that failed, you can try adding this line (above the source command you just added):
                autoload -Uz compinit && compinit
        option 2:
            create a _cortex file in your fpath, for example:
                cortex completion zsh > /usr/local/share/zsh/site-functions/_cortex

Note: this will also add the "cx" alias for cortex for convenience
`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh"},
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_rootCmd.GenBashCompletion(os.Stdout)
			fmt.Print(_bashAliasText)

		case "zsh":
			_rootCmd.GenZshCompletion(os.Stdout)
			fmt.Print("alias cx='cortex'\n\n")

			// https://github.com/spf13/cobra/pull/887
			// https://github.com/corneliusweig/rakkess/blob/master/cmd/completion.go
			// https://github.com/GoogleContainerTools/skaffold/blob/master/cmd/skaffold/app/cmd/completion.go
			// https://github.com/spf13/cobra/issues/881
			// https://github.com/asdf-vm/asdf/issues/266
			fmt.Println("if compquote '' 2>/dev/null; then _cortex; else compdef _cortex cortex; fi")

		default:
			fmt.Println()
			exit.Error(ErrorShellCompletionNotSupported(args[0]))
		}
	},
}
