# ddi-tool

Tool for manipulating [discoverable disk images (DDIs)](https://uapi-group.org/specifications/specs/discoverable_disk_image/) in-place.

Currently, it supports in-place patching of the embedded kernel cmdline in the `.cmdline` section of the UKI.
This can be used to update the expected dm-verity roothash or usrhash after building the image.

## Installation

```shell-session
git clone https://github.com/malt3/ddi-tool && cd ddi-tool
go install .
```

## Usage

```
# build a ddi using systemd-repart
# systemd-repart --json pretty ...
ddi-tool finalize --repart-json repart-output.json --uki-path /EFI/BOOT/BOOTX64.EFI image.raw
```
