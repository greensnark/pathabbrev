# pathabbrev

Abbreviates a directory path, replacing $HOME with ~ and collapsing all other
non-leaf directories starting with an alphabet to their first letter, unless
they appear to contain a version control root (.git, etc.) or an .editorconfig

```sh
$ pathabbrev $HOME
~
$ pathabbrev /
/
$ pathabbrev $HOME/project/.git
~/project/.git
$ pathabbrev $HOME/project/dir
~/project/dir
$ pathabbrev $HOME/other/dir
~/o/dir
```