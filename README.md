# git-cr — Client side encryption for git

[![Build Status](https://travis-ci.org/lucas-clemente/git-cr.svg?branch=master)](https://travis-ci.org/lucas-clemente/git-cr)

## What it does

git-cr is a git remote that encrypts all data in a repo (including metadata) on your client. You can still use all of git's feature, including efficient deltas.

Currently git-cr stores your data in encrypted form in a local directory (e.g. in Dropbox, Google Drive, ...), but a remote backend might be added soon.

## Instructions

### Installation

Installation using go:

```shell
go install github.com/lucas-clemente/git-cr
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
dd if=/dev/random bs=32 count=1 2> /dev/null |base64
```

### Everything else

Just use git!

## How it works

TODO

## Is it secure?

I'm not a cryptographer and git-cr was never audited by anyone. So you probably shouldn't trust it for anything critical. However the cryptography in git-cr is pretty [straightforward](crypto/nacl/nacl.go) and uses [NaCl](http://nacl.cr.yp.to). Check it out!

What git-cr does not hide:

- The size of your deltas (be aware of oracle attacks).
- The dates when you push.

Currently the encryption key is stored in plain text on disk and is visible during some commands, see [#5](https://github.com/lucas-clemente/git-cr/issues/5).

## License

[MIT](LICENSE) of course.
