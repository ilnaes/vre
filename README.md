# vre

vre is a grep-like tool that will visually highlight your regular expression matches.  Input for vre has to be piped in (for now) and matches will be printed out.

For example, to get the first five matches from `infile`, use the command
```sh
./vre < infile | head -n 5
```

Inspired by Junegunn's [fzf](https://github.com/junegunn/fzf)

## Usage

To navigate:

* `<Ctrl-j>` Down
* `<Ctrl-k>` Up
* `<Ctrl-f>` Page down
* `<Ctrl-b>` Page up
* `<Ctrl-c/d>` Cancel without outputting

## Todo 📝 

- [] Match toggling
- [] File inputs
- [] Submatch highlighting
- [] sed-like search/replace

