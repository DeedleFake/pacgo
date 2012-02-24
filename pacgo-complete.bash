#!/bin/bash

. /etc/bash_completion.d/pacman

_pacgo()
{
  local cur=`_get_cword`
  local cmd="${COMP_WORDS[1]}"

  local cmds=(-Syu -Scc -Ss -G
              -Su -Si -S -M)

  case "$COMP_CWORD" in
    1)
      COMPREPLY=($(compgen -W "${cmds[*]}" -- "$cur"))
      ;;
    *)
      case "$cmd" in
        -M)
          _makepkg
          ;;
        -G)
          ;;
        *)
          _pacman
          ;;
      esac
      ;;
  esac
}

complete $filenames -F _pacgo pacgo

# vim:sw=2 ts=2 et
