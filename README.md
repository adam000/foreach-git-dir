## About

NOTE: I am currently transitioning from this single-purpose project to a more general
approach. I'll update the README as it takes shape.

This is a project to help me keep tabs on the leftover work I have in other projects.
Given a directory, check-git-dirs will recurse and find all possible git roots.
It then checks for any outstanding changes (and stashes, if specified) and displays
them to the user.

## Future Work

It would be nice if this tool ran `git fetch --all` and, if the remote has newer
commits, report that as well.

## License

This is licensed under the MIT license. See the LICENSE file for details.
