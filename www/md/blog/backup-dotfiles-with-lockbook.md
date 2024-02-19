# Backup dotfiles with lockbook on macos
Lockbook is a blazingly fast note taking application [written in rust](https://github.com/lockbook/lockbook).
We can store compressed and encrypted text files in lockbook's cloud on the free tier.
Lockbook's CLI supports automatic file syncronization similar to Google Drive.

In this post, I'll show how I back up my [nvchad](https://nvchad.com/) custom config ([docs](https://nvchad.com/docs/config/walkthrough)). This symlink strategy can be used to backup any configs or other text data automatically.

### Setting up the `lockbook` CLI
First install `lockbook` version >= `0.8.5`
```bash
brew tap lockbook/lockbook && brew install lockbook
```
- [Or build from source](https://github.com/lockbook/lockbook/tree/master/docs#cli).
- Guide for tab completions on [bash or zsh](https://github.com/lockbook/lockbook/blob/master/docs/guides/cli-completions.md).
```
lockbook
account      -- account management commands
completions  -- generate completions for a given shell
copy         -- import files from your file system into lockbook
debug        -- investigative commands
delete       -- delete a file
edit         -- edit a document
export       -- export a lockbook file to your file system
fs           -- use your lockbook files with your local filesystem by mounting an NFS drive to /tmp/lockbook
list         -- list files and file information
move         -- move a file to a new parent
new          -- create a new file at the given path or do nothing if it exists
rename       -- rename a file
share        -- sharing related commands
sync         -- sync your local changes back to lockbook servers
```
#### The `lockbook fs` command
Automatically syncs your entire lockbook with a localhost NFS drive in `/tmp/lockbook`
```bash
lockbook fs
```

### Back up our dotfiles
Please note, this guide will replace your existing directory with a symlink. It's highly recommended to backup the existing directory before proceeding to avoid losing configurations.
```bash
cp -r ~/.config/nvim/lua/custom ~/.config/nvim/lua/custom_backup
```
### Copy dotfiles to `/tmp/lockbook`
```bash
mkdir /tmp/lockbook/dotfiles
cp -r ~/.config/nvim/lua/custom/ /tmp/lockbook/nvchad-config
```
### Create the symlink
```bash
ln -s /tmp/lockbook/nvchad-config ~/.config/nvim/lua/custom
```
### Verify the symlink
```bash
ls -l ~/.config/nvim/lua
```
Look for the `custom` entry in the output. It should indicate that it is a symlink (`l` at the beginning of the permissions string) and point to your target directory `/tmp/lockbook/dotfiles/nvchad-config`.
