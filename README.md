pacgo
=====

pacgo is an experimental wrapper for [pacman][pacman] written in [Go][go] and heavily inspired by [packer][packer]. Its (eventual) goal is to be fast and easily modifiable. It supports AUR installation, search, and update checking, as well as AUR dependency handling for [makepkg][makepkg]. It is also capable of downloading and extracting source tarballs from the AUR.

Prerequisites
-------------

 * [pacman][pacman].
 * [Go][go].

Optional Deps
-------------

 * [sudo][sudo].

Installation
------------

The easiest way to install pacgo is using the [package provided in the AUR][aurpkg]. It's also possible to install using the go tool:

> go get github.com/DeedleFake/pacgo

For more information about the go tool, run the following command after installing Go:

> go help

Usage
-----

Usage is much like pacman's, but with a few important differences:

pacgo will not simply pass unrecognized commands through to pacman. pacgo is intended to only wrap commands that have the possibility of using the AUR. pacgo also adds a few new commands. For a complete list, run:

> pacgo --help

Authors
-------

 * [DeedleFake](https://github.com/DeedleFake)

[pacman]: https://wiki.archlinux.org/index.php/Pacman
[makepkg]: https://wiki.archlinux.org/index.php/Makepkg
[packer]: https://github.com/bruenig/packer
[go]: http://www.golang.org
[sudo]: http://www.gratisoft.us/sudo
[aurpkg]: http://aur.archlinux.org/packages.php?ID=56998

<!--
    vim:ts=4 sw=4 et
-->
