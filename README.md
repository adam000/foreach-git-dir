# About

foreach-git-dir is a general-purpose tool for combing through your git repositories.
Take a look at the usage docs with `foreach-git-dir --help` for detailed info.

# Examples

Predicates come before an optional `--` and are filters, and actions are after the `--`
and indicate what output should be done on all git repos that match the predicate(s).
If no action is provided, the full directory is listed.

`foreach-git-dir ~/src [<predicates>] [-- <actions>]`

List all directories that have changes that are not tracked by git:

`foreach-git-dir ~/src -isDirty`

Output `git status` for each directory that is dirty:

`foreach-git-dir ~/src -isDirty -- -status`

Check for GitHub repositories with issues (`gh` command line tool required):

`foreach-git-dir ~/src -hasIssues`

Output all of said issues (`gh` command line tool required):

`foreach-git-dir ~/src -hasIssues -- -issues`

You can chain directives together with `-and` or `-or`, though `-and` is implied:

`foreach-git-dir ~/src -isDirty -or -hasIssues`

`foreach-git-dir ~/src -isDirty -hasIssues`

## Customizing

You can pass the `-custom` flag with a value to either the predicates or the actions.
Custom predicates and actions are run with `sh -c` (or `cmd.exe /C` on Windows) by
default, though the shell can be changed with config (see Config below), so make sure you understand what you are running in those predicates and actions.

For predicates, if the return value is 0, then the repo passes the filter, otherwise it
is filtered out.

Example:

`foreach-git-dir ~/src -custom "pwd | grep -v test" -- -custom "cloc ."`

# Config

Configuration lives in `$XDG_CONFIG_HOME/foreach-git-dir/config.json` and can look like:

```
{
    "rootDir": "/Users/adam/src",
    "shell": ["zsh", "-c"],
    "excludes": [
        "junkdrawer"
    ]
}
```

A configured `rootDir` value means that you don't have to pass it on every invocation.

`shell` is passed to custom predicates and actions.

Excludes are useful for if you have a large subdirectory within your rootDir that you
don't want scanned, and is not a git directory. Such subdirectories can cause the program
to spend a lot of time looking for git repos that don't exist.

All values are optional.

# License

This is licensed under the MIT license. See the LICENSE file for details.
