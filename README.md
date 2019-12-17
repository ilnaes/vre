# vre

vre is a grep-like tool with a terminal UI that will visually highlight your regular expression matches as you type. There is nothing tricky behind the scenes. vre is literally using Go's `regexp` package to do the matching.

For now, input for vre has to be piped in and matches will be printed out. For example, to get the first five matches from `infile`, use the command

```sh
vre < infile | head -n 5
```

<img src="https://raw.githubusercontent.com/ilnaes/i/master/vre.gif" width=640>

Inspired by Junegunn's [fzf](https://github.com/junegunn/fzf).

## Usage

To navigate:

- `<Ctrl-j>` Down
- `<Ctrl-k>` Up
- `<Ctrl-f>` Page down
- `<Ctrl-b>` Page up
- `<Enter>` Quit and output matches
- `<Ctrl-c>`/`<Ctrl-d>` Quit without outputting

## Todo 📝

- [x] Line count
- [ ] File inputs
- [ ] Toggle displaying unmatched lines
- [ ] Command line options like tabstop length, etc.
- [ ] Submatch highlighting
- [ ] sed-like search/replace
