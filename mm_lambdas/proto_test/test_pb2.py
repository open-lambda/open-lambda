# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: test.proto

import sys
_b=sys.version_info[0]<3 and (lambda x:x) or (lambda x:x.encode('latin1'))
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from google.protobuf import reflection as _reflection
from google.protobuf import symbol_database as _symbol_database
from google.protobuf import descriptor_pb2
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor.FileDescriptor(
  name='test.proto',
  package='tutorial',
  syntax='proto2',
  serialized_pb=_b('\n\ntest.proto\x12\x08tutorial\"#\n\x06MyDict\x12\x0c\n\x04turn\x18\x01 \x02(\t\x12\x0b\n\x03pow\x18\x02 \x02(\t')
)
_sym_db.RegisterFileDescriptor(DESCRIPTOR)




_MYDICT = _descriptor.Descriptor(
  name='MyDict',
  full_name='tutorial.MyDict',
  filename=None,
  file=DESCRIPTOR,
  containing_type=None,
  fields=[
    _descriptor.FieldDescriptor(
      name='turn', full_name='tutorial.MyDict.turn', index=0,
      number=1, type=9, cpp_type=9, label=2,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None),
    _descriptor.FieldDescriptor(
      name='pow', full_name='tutorial.MyDict.pow', index=1,
      number=2, type=9, cpp_type=9, label=2,
      has_default_value=False, default_value=_b("").decode('utf-8'),
      message_type=None, enum_type=None, containing_type=None,
      is_extension=False, extension_scope=None,
      options=None),
  ],
  extensions=[
  ],
  nested_types=[],
  enum_types=[
  ],
  options=None,
  is_extendable=False,
  syntax='proto2',
  extension_ranges=[],
  oneofs=[
  ],
  serialized_start=24,
  serialized_end=59,
)

DESCRIPTOR.message_types_by_name['MyDict'] = _MYDICT

MyDict = _reflection.GeneratedProtocolMessageType('MyDict', (_message.Message,), dict(
  DESCRIPTOR = _MYDICT,
  __module__ = 'test_pb2'
  # @@protoc_insertion_point(class_scope:tutorial.MyDict)
  ))
_sym_db.RegisterMessage(MyDict)


# @@protoc_insertion_point(module_scope)
