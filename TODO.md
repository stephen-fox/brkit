# TODO

## pattern

- add read method to debruijn object

## iokit

- add readn to buffer object

## memory

- pointer maker from raw bytes source endianess check results
  in wrong endianess for example parsing the following value with
  `pm.FromRawBytesOrExit(tlsRaw, binary.LittleEndian)` results in
  `0x40f7ddf7ff7f0000`

```
process: Read:
00000000  40 f7 dd f7 ff 7f                                 |@.....|
tls: 0x40f7ddf7ff7f0000
```
