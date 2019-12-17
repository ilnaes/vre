# vre

vre is a grep-like tool with a terminal UI that will visually highlight your regular expression matches as you type. There is nothing tricky behind the scenes. vre is just using Go's `regexp` package to do the matching.

<img src="https://raw.githubusercontent.com/ilnaes/i/master/vre.gif" width=640>

Inspired by Junegunn's [fzf](https://github.com/junegunn/fzf).

## Usage

One can either pipe in the data to be matched or feed in the files via command line arguments. For example, to run vre on all go files in the `internal` director, use the command

```sh
vre internal/*.go
```

To print the first five matches from `ls`, use the command

```sh
ls | vre | head -n 5
```

To navigate:

- `<Ctrl-j>` Down
- `<Ctrl-k>` Up
- `<Ctrl-f>` Page down
- `<Ctrl-b>` Page up
- `<Enter>` Quit and output matches
- `<Ctrl-c>`/`<Ctrl-d>` Quit without outputting

## Todo 📝

- [x] Line count
- [x] File inputs
- [ ] Toggle displaying unmatched lines
- [ ] Command line options like tabstop length, etc.
- [ ] Submatch highlighting
- [ ] sed-like search/replace
