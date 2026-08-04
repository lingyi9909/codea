#!/usr/bin/env python3
"""Count DNS and HTTP(S) packets in classic pcap files without third-party packages."""

from __future__ import annotations

import pathlib
import struct
import sys


def ip_offset(packet: bytes, linktype: int) -> int | None:
    if linktype == 1:  # Ethernet
        if len(packet) < 14:
            return None
        offset = 14
        ether_type = struct.unpack_from("!H", packet, 12)[0]
        while ether_type in {0x8100, 0x88A8, 0x9100}:
            if len(packet) < offset + 4:
                return None
            ether_type = struct.unpack_from("!H", packet, offset + 2)[0]
            offset += 4
        return offset if ether_type in {0x0800, 0x86DD} else None
    if linktype in {0, 108}:  # BSD loopback/null
        return 4 if len(packet) >= 5 and packet[4] >> 4 in {4, 6} else None
    if linktype == 101:  # Raw IP
        return 0 if packet and packet[0] >> 4 in {4, 6} else None
    if linktype == 113:  # Linux cooked v1
        return 16 if len(packet) >= 17 and packet[16] >> 4 in {4, 6} else None
    if linktype == 276:  # Linux cooked v2
        return 20 if len(packet) >= 21 and packet[20] >> 4 in {4, 6} else None
    raise ValueError(f"unsupported pcap link type: {linktype}")


def transport(packet: bytes, offset: int) -> tuple[int, int] | None:
    version = packet[offset] >> 4
    if version == 4:
        if len(packet) < offset + 20:
            return None
        header_length = (packet[offset] & 0x0F) * 4
        protocol = packet[offset + 9]
        transport_offset = offset + header_length
    elif version == 6:
        if len(packet) < offset + 40:
            return None
        protocol = packet[offset + 6]
        transport_offset = offset + 40
        while protocol in {0, 43, 44, 60}:
            if len(packet) < transport_offset + 8:
                return None
            next_protocol = packet[transport_offset]
            if protocol == 44:
                extension_length = 8
            else:
                extension_length = (packet[transport_offset + 1] + 1) * 8
            transport_offset += extension_length
            protocol = next_protocol
    else:
        return None

    if protocol not in {6, 17} or len(packet) < transport_offset + 4:
        return None
    source_port, destination_port = struct.unpack_from("!HH", packet, transport_offset)
    return source_port, destination_port


def summarize(path: pathlib.Path, start: float, end: float) -> tuple[int, int, int]:
    data = path.read_bytes()
    if len(data) < 24:
        raise ValueError("pcap global header is truncated")
    magic = data[:4]
    nanoseconds = magic in {b"\x4d\x3c\xb2\xa1", b"\xa1\xb2\x3c\x4d"}
    if magic in {b"\xd4\xc3\xb2\xa1", b"\x4d\x3c\xb2\xa1"}:
        endian = "<"
    elif magic in {b"\xa1\xb2\xc3\xd4", b"\xa1\xb2\x3c\x4d"}:
        endian = ">"
    else:
        raise ValueError("unsupported pcap magic")
    linktype = struct.unpack_from(f"{endian}I", data, 20)[0]

    dns = web = packets = 0
    cursor = 24
    while cursor < len(data):
        if len(data) - cursor < 16:
            raise ValueError("pcap packet header is truncated")
        seconds, fraction, included_length = struct.unpack_from(f"{endian}III", data, cursor)
        cursor += 16
        packet = data[cursor : cursor + included_length]
        if len(packet) != included_length:
            raise ValueError("pcap packet data is truncated")
        cursor += included_length
        timestamp = seconds + fraction / (1_000_000_000 if nanoseconds else 1_000_000)
        if timestamp < start or timestamp > end:
            continue
        packets += 1
        offset = ip_offset(packet, linktype)
        ports = transport(packet, offset) if offset is not None else None
        if ports and 53 in ports:
            dns += 1
        if ports and ({80, 443} & set(ports)):
            web += 1
    return dns, web, packets


def main() -> int:
    if len(sys.argv) not in {2, 4}:
        print(f"Usage: {sys.argv[0]} <pcap> [start-epoch end-epoch]", file=sys.stderr)
        return 2
    start, end = (float(sys.argv[2]), float(sys.argv[3])) if len(sys.argv) == 4 else (0.0, float("inf"))
    try:
        dns, web, packets = summarize(pathlib.Path(sys.argv[1]), start, end)
    except Exception as exc:
        print(f"pcap parse failed: {exc}", file=sys.stderr)
        return 2
    print(dns, web, packets)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
