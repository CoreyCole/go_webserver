# Backup dotfiles with lockbook on macos
Lockbook is a 100% [open source](https://github.com/lockbook/lockbook) encrypted note taking application written in rust.
We can store highly-compressed text files in their cloud on the free tier.
Lockbook's CLI supports automatic file syncronization similar to Google Drive.

In this post, I'll show how I back up my [nvchad](https://nvchad.com/) custom config ([docs](https://nvchad.com/docs/config/walkthrough)). This symlink strategy can be used to backup any configs or other text data automatically.

### Setting up the CLI
First install
```bash
brew tap lockbook/lockbook && brew install lockbook
```
- [Or build from source](https://github.com/lockbook/lockbook/tree/master/docs#cli).
- enable tab completions on [bash or zsh](https://github.com/lockbook/lockbook/blob/master/docs/guides/cli-completions.md) in the docs.

### Backing up our dotfiles
Please note, this operation will replace your existing directory with a symlink. It's highly recommended to backup the existing directory before proceeding to avoid losing configurations.
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
