// Code generated by protoc-gen-go.
// source: login.proto
// DO NOT EDIT!

/*
Package proto is a generated protocol buffer package.

It is generated from these files:
	login.proto

It has these top-level messages:
	User
*/
package proto

import proto1 "github.com/golang/protobuf/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal

type User struct {
}

func (m *User) Reset()         { *m = User{} }
func (m *User) String() string { return proto1.CompactTextString(m) }
func (*User) ProtoMessage()    {}

type User_Login struct {
	Uuid      string `protobuf:"bytes,1,opt,name=uuid" json:"uuid,omitempty"`
	Host      string `protobuf:"bytes,2,opt,name=host" json:"host,omitempty"`
	LoginType int32  `protobuf:"varint,3,opt,name=login_type" json:"login_type,omitempty"`
}

func (m *User_Login) Reset()         { *m = User_Login{} }
func (m *User_Login) String() string { return proto1.CompactTextString(m) }
func (*User_Login) ProtoMessage()    {}

type User_Uid struct {
	Uid int32 `protobuf:"varint,1,opt,name=uid" json:"uid,omitempty"`
}

func (m *User_Uid) Reset()         { *m = User_Uid{} }
func (m *User_Uid) String() string { return proto1.CompactTextString(m) }
func (*User_Uid) ProtoMessage()    {}

type User_Auth struct {
	Uid   int32 `protobuf:"varint,1,opt,name=uid" json:"uid,omitempty"`
	Stats int32 `protobuf:"varint,2,opt,name=stats" json:"stats,omitempty"`
	Code  int32 `protobuf:"varint,3,opt,name=code" json:"code,omitempty"`
}

func (m *User_Auth) Reset()         { *m = User_Auth{} }
func (m *User_Auth) String() string { return proto1.CompactTextString(m) }
func (*User_Auth) ProtoMessage()    {}

type User_NullResult struct {
}

func (m *User_NullResult) Reset()         { *m = User_NullResult{} }
func (m *User_NullResult) String() string { return proto1.CompactTextString(m) }
func (*User_NullResult) ProtoMessage()    {}

func init() {
}
