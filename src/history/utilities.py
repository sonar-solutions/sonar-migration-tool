import json
from google.protobuf.internal.encoder import _VarintBytes
from google.protobuf import json_format


def append_protobuf_to_file(file_path, payload, proto_class):
    proto = convert_to_protobuf(payload=payload, proto_class=proto_class)
    with open(file_path, 'ab') as f:
        f.write(_VarintBytes(proto.ByteSize()))
        f.write(proto.SerializeToString())


def convert_to_protobuf(payload, proto_class):
    instance = proto_class()
    json_format.Parse(json.dumps(payload), instance)
    return instance


def write_protobuf_to_file(filepath, payload, proto_class):
    proto = convert_to_protobuf(payload=payload, proto_class=proto_class)
    with open(filepath, 'wb') as f:
        f.write(proto.SerializeToString())
