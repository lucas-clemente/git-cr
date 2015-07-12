# 🔒 git-cr — Client side encryption for git

[![Build Status](https://travis-ci.org/lucas-clemente/git-cr.svg?branch=master)](https://travis-ci.org/lucas-clemente/git-cr)

## What it does

git-cr is a git remote that encrypts all data in a repo (including metadata) client-side. You can still use all of git's feature, including efficient deltas.

Currently git-cr stores your data in encrypted form in a local directory (e.g. in Dropbox, Google Drive, …), but a remote backend might be added soon.

## What's new about git-cr

There are some [tools](https://github.com/shadowhand/git-encrypt) and [tutorials](https://gist.github.com/shadowhand/873637) on how to encrypt single files stored in git. git-cr is different: it encrypts the whole repo, including metadata such as file names, branch names, commit messages. You also don't loose as many git features (e.g. awesome compression and efficient pushes / pulls).

## Instructions

### Installation

Installation using go:

```shell
go get github.com/lucas-clemente/git-cr
```

Alternatively (if you don't have go), you can download a current release from [github](https://github.com/lucas-clemente/git-cr/releases) and move it somewhere into your `$PATH`.

### Cloning

To clone an existing repo:

```shell
git cr clone /path/to/git-cr/repo nacl:MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI= my-clone
```

### Pushing

```shell
git cr add crypto /path/to/git-cr/repo nacl:MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=
git push crypto master
```

### Encryption

The secret for NaCl is a 32 byte base64 encoded string. You can generate a new secret using

```shell
echo -n nacl:; head -c32 /dev/urandom |base64
```

### Everything else

Just use git!

## How it works

git-cr uses a git feature called [external remotes](http://git-scm.com/docs/git-remote-ext):

```shell
$ git remote -v
crypto	ext::git cr %G run /path/to/remote nacl:MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI= (fetch)
crypto	ext::git cr %G run /path/to/remote nacl:MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI= (push)
```

Any git operation that needs the remote (e.g. pull, push, clone) then starts git-cr as a child process and uses pipes to talk the git protocol.

git-cr manages two things, refs (i.e. branch names) and packfiles (i.e. your data), in numbered _revisions_. Each push creates a new revision. These revisions are never visible to git in any way!

When pushing, git first sends the ref updates that git-cr uses to create a new revision. Then git sends the diffs as a so-called _thin packfile_, that git-cr encrypts and stores.

When pulling, git and git-cr first work out the current state of the local git repo. git-cr calculates the minimum set of previously stored packfiles it needs to send (i.e. all packfiles since the last revision the client completely has). Then it decrypts these packfiles, merges them into one and sends it to git.

## Is it secure?

I'm not a cryptographer and git-cr was never audited by anyone. So you probably shouldn't trust it for anything critical.

git-cr uses the backend to store whole files only. Files can either be git packfiles, or a manifest file containing the git refs for each revision. Each file is encrypted using [NaCl's](http://nacl.cr.yp.to) authenticated encryption `crypto_secretbox`. The key is static and part of the repository URL, while the nonce is generated (using `crypto/rand`) per file and stored prepended to the ciphertext.

The source code for this can be found [here](crypto/nacl/nacl.go). Check it out!

What git-cr does not hide:

- The size of your deltas (be aware of oracle attacks).
- The dates when you push.

Currently the encryption key is stored in plain text on disk and is visible during some commands, see [#5](https://github.com/lucas-clemente/git-cr/issues/5).

## License

[MIT](LICENSE) of course.
