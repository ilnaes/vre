# vre

vre is a grep-like tool that will visually highlight your regular expression matches.  Input for vre has to be piped in (for now) and matches will be printed out.

For example, to get the first five matches from `infile`, use the command
```sh
./vre < infile | head -n 5
```

Inspired by Junegunn's [fzf](https://github.com/junegunn/fzf)
