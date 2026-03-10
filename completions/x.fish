function __complete_x
    set -lx COMP_LINE (commandline -cp)
    test -z (commandline -ct)
    and set COMP_LINE "$COMP_LINE "
    /Users/developer/Documents/GitHub/workspaces/CLI-Tools/Tools/x-cli/x
end
complete -f -c x -a "(__complete_x)"
