// Auto-modified by gostd at 2020-11-26T19:02:21.208975196-05:00 by version 0.1

package core

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"container/heap"
	"container/list"
	"container/ring"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rc4"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"database/sql/driver"
	"debug/dwarf"
	"debug/elf"
	"debug/gosym"
	"debug/macho"
	"debug/pe"
	"debug/plan9obj"
	"encoding"
	"encoding/ascii85"
	"encoding/asn1"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"expvar"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/constant"
	"go/doc"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/scanner"
	"go/token"
	"go/types"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"hash/maphash"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"index/suffixarray"
	"io"
	"log"
	"log/syslog"
	"math/big"
	"math/rand"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/http/cgi"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httptrace"
	"net/http/httputil"
	"net/mail"
	"net/rpc"
	"net/smtp"
	"net/textproto"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"plugin"
	"reflect"
	"regexp"
	"regexp/syntax"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"testing/quick"
	text_scanner "text/scanner"
	"text/tabwriter"
	text_template "text/template"
	"text/template/parse"
	"time"
	"unicode"
	"unsafe"
)

var GoTypesVec [978]*GoTypeInfo

func SwitchGoType(g interface{}) int {
	switch g.(type) {
	case tar.Format:
		return 0
	case tar.Header:
		return 1
	case tar.Reader:
		return 2
	case tar.Writer:
		return 3
	case zip.Compressor:
		return 4
	case zip.Decompressor:
		return 5
	case zip.File:
		return 6
	case zip.FileHeader:
		return 7
	case zip.ReadCloser:
		return 8
	case zip.Reader:
		return 9
	case zip.Writer:
		return 10
	case bufio.ReadWriter:
		return 11
	case bufio.Reader:
		return 12
	case bufio.Scanner:
		return 13
	case bufio.SplitFunc:
		return 14
	case bufio.Writer:
		return 15
	case bytes.Buffer:
		return 16
	case bytes.Reader:
		return 17
	case bzip2.StructuralError:
		return 18
	case flate.CorruptInputError:
		return 19
	case flate.InternalError:
		return 20
	case flate.ReadError:
		return 21
	case flate.WriteError:
		return 22
	case flate.Writer:
		return 23
	case gzip.Header:
		return 24
	case gzip.Reader:
		return 25
	case gzip.Writer:
		return 26
	case lzw.Order:
		return 27
	case zlib.Writer:
		return 28
	case list.Element:
		return 29
	case list.List:
		return 30
	case ring.Ring:
		return 31
	case context.CancelFunc:
		return 32
	case crypto.Hash:
		return 33
	case aes.KeySizeError:
		return 34
	case cipher.StreamReader:
		return 35
	case cipher.StreamWriter:
		return 36
	case des.KeySizeError:
		return 37
	case dsa.ParameterSizes:
		return 38
	case dsa.Parameters:
		return 39
	case dsa.PrivateKey:
		return 40
	case dsa.PublicKey:
		return 41
	case ecdsa.PrivateKey:
		return 42
	case ecdsa.PublicKey:
		return 43
	case ed25519.PrivateKey:
		return 44
	case ed25519.PublicKey:
		return 45
	case elliptic.CurveParams:
		return 46
	case rc4.Cipher:
		return 47
	case rc4.KeySizeError:
		return 48
	case rsa.CRTValue:
		return 49
	case rsa.OAEPOptions:
		return 50
	case rsa.PKCS1v15DecryptOptions:
		return 51
	case rsa.PSSOptions:
		return 52
	case rsa.PrecomputedValues:
		return 53
	case rsa.PrivateKey:
		return 54
	case rsa.PublicKey:
		return 55
	case tls.Certificate:
		return 56
	case tls.CertificateRequestInfo:
		return 57
	case tls.CipherSuite:
		return 58
	case tls.ClientAuthType:
		return 59
	case tls.ClientHelloInfo:
		return 60
	case tls.ClientSessionState:
		return 61
	case tls.Config:
		return 62
	case tls.Conn:
		return 63
	case tls.ConnectionState:
		return 64
	case tls.CurveID:
		return 65
	case tls.Dialer:
		return 66
	case tls.RecordHeaderError:
		return 67
	case tls.RenegotiationSupport:
		return 68
	case tls.SignatureScheme:
		return 69
	case x509.CertPool:
		return 70
	case x509.Certificate:
		return 71
	case x509.CertificateInvalidError:
		return 72
	case x509.CertificateRequest:
		return 73
	case x509.ConstraintViolationError:
		return 74
	case x509.ExtKeyUsage:
		return 75
	case x509.HostnameError:
		return 76
	case x509.InsecureAlgorithmError:
		return 77
	case x509.InvalidReason:
		return 78
	case x509.KeyUsage:
		return 79
	case x509.PEMCipher:
		return 80
	case x509.PublicKeyAlgorithm:
		return 81
	case x509.RevocationList:
		return 82
	case x509.SignatureAlgorithm:
		return 83
	case x509.SystemRootsError:
		return 84
	case x509.UnhandledCriticalExtension:
		return 85
	case x509.UnknownAuthorityError:
		return 86
	case x509.VerifyOptions:
		return 87
	case pkix.AlgorithmIdentifier:
		return 88
	case pkix.AttributeTypeAndValue:
		return 89
	case pkix.AttributeTypeAndValueSET:
		return 90
	case pkix.CertificateList:
		return 91
	case pkix.Extension:
		return 92
	case pkix.Name:
		return 93
	case pkix.RDNSequence:
		return 94
	case pkix.RelativeDistinguishedNameSET:
		return 95
	case pkix.RevokedCertificate:
		return 96
	case pkix.TBSCertificateList:
		return 97
	case sql.ColumnType:
		return 98
	case sql.Conn:
		return 99
	case sql.DB:
		return 100
	case sql.DBStats:
		return 101
	case sql.IsolationLevel:
		return 102
	case sql.NamedArg:
		return 103
	case sql.NullBool:
		return 104
	case sql.NullFloat64:
		return 105
	case sql.NullInt32:
		return 106
	case sql.NullInt64:
		return 107
	case sql.NullString:
		return 108
	case sql.NullTime:
		return 109
	case sql.Out:
		return 110
	case sql.RawBytes:
		return 111
	case sql.Row:
		return 112
	case sql.Rows:
		return 113
	case sql.Stmt:
		return 114
	case sql.Tx:
		return 115
	case sql.TxOptions:
		return 116
	case driver.IsolationLevel:
		return 117
	case driver.NamedValue:
		return 118
	case driver.NotNull:
		return 119
	case driver.Null:
		return 120
	case driver.RowsAffected:
		return 121
	case driver.TxOptions:
		return 122
	case dwarf.AddrType:
		return 123
	case dwarf.ArrayType:
		return 124
	case dwarf.Attr:
		return 125
	case dwarf.BasicType:
		return 126
	case dwarf.BoolType:
		return 127
	case dwarf.CharType:
		return 128
	case dwarf.Class:
		return 129
	case dwarf.CommonType:
		return 130
	case dwarf.ComplexType:
		return 131
	case dwarf.Data:
		return 132
	case dwarf.DecodeError:
		return 133
	case dwarf.DotDotDotType:
		return 134
	case dwarf.Entry:
		return 135
	case dwarf.EnumType:
		return 136
	case dwarf.EnumValue:
		return 137
	case dwarf.Field:
		return 138
	case dwarf.FloatType:
		return 139
	case dwarf.FuncType:
		return 140
	case dwarf.IntType:
		return 141
	case dwarf.LineEntry:
		return 142
	case dwarf.LineFile:
		return 143
	case dwarf.LineReader:
		return 144
	case dwarf.LineReaderPos:
		return 145
	case dwarf.Offset:
		return 146
	case dwarf.PtrType:
		return 147
	case dwarf.QualType:
		return 148
	case dwarf.Reader:
		return 149
	case dwarf.StructField:
		return 150
	case dwarf.StructType:
		return 151
	case dwarf.Tag:
		return 152
	case dwarf.TypedefType:
		return 153
	case dwarf.UcharType:
		return 154
	case dwarf.UintType:
		return 155
	case dwarf.UnspecifiedType:
		return 156
	case dwarf.UnsupportedType:
		return 157
	case dwarf.VoidType:
		return 158
	case elf.Chdr32:
		return 159
	case elf.Chdr64:
		return 160
	case elf.Class:
		return 161
	case elf.CompressionType:
		return 162
	case elf.Data:
		return 163
	case elf.Dyn32:
		return 164
	case elf.Dyn64:
		return 165
	case elf.DynFlag:
		return 166
	case elf.DynTag:
		return 167
	case elf.File:
		return 168
	case elf.FileHeader:
		return 169
	case elf.FormatError:
		return 170
	case elf.Header32:
		return 171
	case elf.Header64:
		return 172
	case elf.ImportedSymbol:
		return 173
	case elf.Machine:
		return 174
	case elf.NType:
		return 175
	case elf.OSABI:
		return 176
	case elf.Prog:
		return 177
	case elf.Prog32:
		return 178
	case elf.Prog64:
		return 179
	case elf.ProgFlag:
		return 180
	case elf.ProgHeader:
		return 181
	case elf.ProgType:
		return 182
	case elf.R_386:
		return 183
	case elf.R_390:
		return 184
	case elf.R_AARCH64:
		return 185
	case elf.R_ALPHA:
		return 186
	case elf.R_ARM:
		return 187
	case elf.R_MIPS:
		return 188
	case elf.R_PPC:
		return 189
	case elf.R_PPC64:
		return 190
	case elf.R_RISCV:
		return 191
	case elf.R_SPARC:
		return 192
	case elf.R_X86_64:
		return 193
	case elf.Rel32:
		return 194
	case elf.Rel64:
		return 195
	case elf.Rela32:
		return 196
	case elf.Rela64:
		return 197
	case elf.Section:
		return 198
	case elf.Section32:
		return 199
	case elf.Section64:
		return 200
	case elf.SectionFlag:
		return 201
	case elf.SectionHeader:
		return 202
	case elf.SectionIndex:
		return 203
	case elf.SectionType:
		return 204
	case elf.Sym32:
		return 205
	case elf.Sym64:
		return 206
	case elf.SymBind:
		return 207
	case elf.SymType:
		return 208
	case elf.SymVis:
		return 209
	case elf.Symbol:
		return 210
	case elf.Type:
		return 211
	case elf.Version:
		return 212
	case gosym.DecodingError:
		return 213
	case gosym.Func:
		return 214
	case gosym.LineTable:
		return 215
	case gosym.Obj:
		return 216
	case gosym.Sym:
		return 217
	case gosym.Table:
		return 218
	case gosym.UnknownFileError:
		return 219
	case gosym.UnknownLineError:
		return 220
	case macho.Cpu:
		return 221
	case macho.Dylib:
		return 222
	case macho.DylibCmd:
		return 223
	case macho.Dysymtab:
		return 224
	case macho.DysymtabCmd:
		return 225
	case macho.FatArch:
		return 226
	case macho.FatArchHeader:
		return 227
	case macho.FatFile:
		return 228
	case macho.File:
		return 229
	case macho.FileHeader:
		return 230
	case macho.FormatError:
		return 231
	case macho.LoadBytes:
		return 232
	case macho.LoadCmd:
		return 233
	case macho.Nlist32:
		return 234
	case macho.Nlist64:
		return 235
	case macho.Regs386:
		return 236
	case macho.RegsAMD64:
		return 237
	case macho.Reloc:
		return 238
	case macho.RelocTypeARM:
		return 239
	case macho.RelocTypeARM64:
		return 240
	case macho.RelocTypeGeneric:
		return 241
	case macho.RelocTypeX86_64:
		return 242
	case macho.Rpath:
		return 243
	case macho.RpathCmd:
		return 244
	case macho.Section:
		return 245
	case macho.Section32:
		return 246
	case macho.Section64:
		return 247
	case macho.SectionHeader:
		return 248
	case macho.Segment:
		return 249
	case macho.Segment32:
		return 250
	case macho.Segment64:
		return 251
	case macho.SegmentHeader:
		return 252
	case macho.Symbol:
		return 253
	case macho.Symtab:
		return 254
	case macho.SymtabCmd:
		return 255
	case macho.Thread:
		return 256
	case macho.Type:
		return 257
	case pe.COFFSymbol:
		return 258
	case pe.DataDirectory:
		return 259
	case pe.File:
		return 260
	case pe.FileHeader:
		return 261
	case pe.FormatError:
		return 262
	case pe.ImportDirectory:
		return 263
	case pe.OptionalHeader32:
		return 264
	case pe.OptionalHeader64:
		return 265
	case pe.Reloc:
		return 266
	case pe.Section:
		return 267
	case pe.SectionHeader:
		return 268
	case pe.SectionHeader32:
		return 269
	case pe.StringTable:
		return 270
	case pe.Symbol:
		return 271
	case plan9obj.File:
		return 272
	case plan9obj.FileHeader:
		return 273
	case plan9obj.Section:
		return 274
	case plan9obj.SectionHeader:
		return 275
	case plan9obj.Sym:
		return 276
	case ascii85.CorruptInputError:
		return 277
	case asn1.BitString:
		return 278
	case asn1.Enumerated:
		return 279
	case asn1.Flag:
		return 280
	case asn1.ObjectIdentifier:
		return 281
	case asn1.RawContent:
		return 282
	case asn1.RawValue:
		return 283
	case asn1.StructuralError:
		return 284
	case asn1.SyntaxError:
		return 285
	case base32.CorruptInputError:
		return 286
	case base32.Encoding:
		return 287
	case base64.CorruptInputError:
		return 288
	case base64.Encoding:
		return 289
	case csv.ParseError:
		return 290
	case csv.Reader:
		return 291
	case csv.Writer:
		return 292
	case gob.CommonType:
		return 293
	case gob.Decoder:
		return 294
	case gob.Encoder:
		return 295
	case hex.InvalidByteError:
		return 296
	case json.Decoder:
		return 297
	case json.Delim:
		return 298
	case json.Encoder:
		return 299
	case json.InvalidUTF8Error:
		return 300
	case json.InvalidUnmarshalError:
		return 301
	case json.MarshalerError:
		return 302
	case json.Number:
		return 303
	case json.RawMessage:
		return 304
	case json.SyntaxError:
		return 305
	case json.UnmarshalFieldError:
		return 306
	case json.UnmarshalTypeError:
		return 307
	case json.UnsupportedTypeError:
		return 308
	case json.UnsupportedValueError:
		return 309
	case pem.Block:
		return 310
	case xml.Attr:
		return 311
	case xml.CharData:
		return 312
	case xml.Comment:
		return 313
	case xml.Decoder:
		return 314
	case xml.Directive:
		return 315
	case xml.Encoder:
		return 316
	case xml.EndElement:
		return 317
	case xml.Name:
		return 318
	case xml.ProcInst:
		return 319
	case xml.StartElement:
		return 320
	case xml.SyntaxError:
		return 321
	case xml.TagPathError:
		return 322
	case xml.UnmarshalError:
		return 323
	case xml.UnsupportedTypeError:
		return 324
	case expvar.Float:
		return 325
	case expvar.Func:
		return 326
	case expvar.Int:
		return 327
	case expvar.KeyValue:
		return 328
	case expvar.Map:
		return 329
	case expvar.String:
		return 330
	case flag.ErrorHandling:
		return 331
	case flag.Flag:
		return 332
	case flag.FlagSet:
		return 333
	case ast.ArrayType:
		return 334
	case ast.AssignStmt:
		return 335
	case ast.BadDecl:
		return 336
	case ast.BadExpr:
		return 337
	case ast.BadStmt:
		return 338
	case ast.BasicLit:
		return 339
	case ast.BinaryExpr:
		return 340
	case ast.BlockStmt:
		return 341
	case ast.BranchStmt:
		return 342
	case ast.CallExpr:
		return 343
	case ast.CaseClause:
		return 344
	case ast.ChanDir:
		return 345
	case ast.ChanType:
		return 346
	case ast.CommClause:
		return 347
	case ast.Comment:
		return 348
	case ast.CommentGroup:
		return 349
	case ast.CommentMap:
		return 350
	case ast.CompositeLit:
		return 351
	case ast.DeclStmt:
		return 352
	case ast.DeferStmt:
		return 353
	case ast.Ellipsis:
		return 354
	case ast.EmptyStmt:
		return 355
	case ast.ExprStmt:
		return 356
	case ast.Field:
		return 357
	case ast.FieldFilter:
		return 358
	case ast.FieldList:
		return 359
	case ast.File:
		return 360
	case ast.Filter:
		return 361
	case ast.ForStmt:
		return 362
	case ast.FuncDecl:
		return 363
	case ast.FuncLit:
		return 364
	case ast.FuncType:
		return 365
	case ast.GenDecl:
		return 366
	case ast.GoStmt:
		return 367
	case ast.Ident:
		return 368
	case ast.IfStmt:
		return 369
	case ast.ImportSpec:
		return 370
	case ast.Importer:
		return 371
	case ast.IncDecStmt:
		return 372
	case ast.IndexExpr:
		return 373
	case ast.InterfaceType:
		return 374
	case ast.KeyValueExpr:
		return 375
	case ast.LabeledStmt:
		return 376
	case ast.MapType:
		return 377
	case ast.MergeMode:
		return 378
	case ast.ObjKind:
		return 379
	case ast.Object:
		return 380
	case ast.Package:
		return 381
	case ast.ParenExpr:
		return 382
	case ast.RangeStmt:
		return 383
	case ast.ReturnStmt:
		return 384
	case ast.Scope:
		return 385
	case ast.SelectStmt:
		return 386
	case ast.SelectorExpr:
		return 387
	case ast.SendStmt:
		return 388
	case ast.SliceExpr:
		return 389
	case ast.StarExpr:
		return 390
	case ast.StructType:
		return 391
	case ast.SwitchStmt:
		return 392
	case ast.TypeAssertExpr:
		return 393
	case ast.TypeSpec:
		return 394
	case ast.TypeSwitchStmt:
		return 395
	case ast.UnaryExpr:
		return 396
	case ast.ValueSpec:
		return 397
	case build.Context:
		return 398
	case build.ImportMode:
		return 399
	case build.MultiplePackageError:
		return 400
	case build.NoGoError:
		return 401
	case build.Package:
		return 402
	case constant.Kind:
		return 403
	case doc.Example:
		return 404
	case doc.Filter:
		return 405
	case doc.Func:
		return 406
	case doc.Mode:
		return 407
	case doc.Note:
		return 408
	case doc.Package:
		return 409
	case doc.Type:
		return 410
	case doc.Value:
		return 411
	case importer.Lookup:
		return 412
	case parser.Mode:
		return 413
	case printer.CommentedNode:
		return 414
	case printer.Config:
		return 415
	case printer.Mode:
		return 416
	case scanner.Error:
		return 417
	case scanner.ErrorHandler:
		return 418
	case scanner.ErrorList:
		return 419
	case scanner.Mode:
		return 420
	case scanner.Scanner:
		return 421
	case token.File:
		return 422
	case token.FileSet:
		return 423
	case token.Pos:
		return 424
	case token.Position:
		return 425
	case token.Token:
		return 426
	case types.Array:
		return 427
	case types.Basic:
		return 428
	case types.BasicInfo:
		return 429
	case types.BasicKind:
		return 430
	case types.Builtin:
		return 431
	case types.Chan:
		return 432
	case types.ChanDir:
		return 433
	case types.Checker:
		return 434
	case types.Config:
		return 435
	case types.Const:
		return 436
	case types.Error:
		return 437
	case types.Func:
		return 438
	case types.ImportMode:
		return 439
	case types.Info:
		return 440
	case types.Initializer:
		return 441
	case types.Interface:
		return 442
	case types.Label:
		return 443
	case types.Map:
		return 444
	case types.MethodSet:
		return 445
	case types.Named:
		return 446
	case types.Nil:
		return 447
	case types.Package:
		return 448
	case types.PkgName:
		return 449
	case types.Pointer:
		return 450
	case types.Qualifier:
		return 451
	case types.Scope:
		return 452
	case types.Selection:
		return 453
	case types.SelectionKind:
		return 454
	case types.Signature:
		return 455
	case types.Slice:
		return 456
	case types.StdSizes:
		return 457
	case types.Struct:
		return 458
	case types.Tuple:
		return 459
	case types.TypeAndValue:
		return 460
	case types.TypeName:
		return 461
	case types.Var:
		return 462
	case crc32.Table:
		return 463
	case crc64.Table:
		return 464
	case maphash.Hash:
		return 465
	case maphash.Seed:
		return 466
	case template.CSS:
		return 467
	case template.Error:
		return 468
	case template.ErrorCode:
		return 469
	case template.FuncMap:
		return 470
	case template.HTML:
		return 471
	case template.HTMLAttr:
		return 472
	case template.JS:
		return 473
	case template.JSStr:
		return 474
	case template.Srcset:
		return 475
	case template.Template:
		return 476
	case template.URL:
		return 477
	case image.Alpha:
		return 478
	case image.Alpha16:
		return 479
	case image.CMYK:
		return 480
	case image.Config:
		return 481
	case image.Gray:
		return 482
	case image.Gray16:
		return 483
	case image.NRGBA:
		return 484
	case image.NRGBA64:
		return 485
	case image.NYCbCrA:
		return 486
	case image.Paletted:
		return 487
	case image.Point:
		return 488
	case image.RGBA:
		return 489
	case image.RGBA64:
		return 490
	case image.Rectangle:
		return 491
	case image.Uniform:
		return 492
	case image.YCbCr:
		return 493
	case image.YCbCrSubsampleRatio:
		return 494
	case color.Alpha:
		return 495
	case color.Alpha16:
		return 496
	case color.CMYK:
		return 497
	case color.Gray:
		return 498
	case color.Gray16:
		return 499
	case color.NRGBA:
		return 500
	case color.NRGBA64:
		return 501
	case color.NYCbCrA:
		return 502
	case color.Palette:
		return 503
	case color.RGBA:
		return 504
	case color.RGBA64:
		return 505
	case color.YCbCr:
		return 506
	case draw.Op:
		return 507
	case gif.GIF:
		return 508
	case gif.Options:
		return 509
	case jpeg.FormatError:
		return 510
	case jpeg.Options:
		return 511
	case jpeg.UnsupportedError:
		return 512
	case png.CompressionLevel:
		return 513
	case png.Encoder:
		return 514
	case png.EncoderBuffer:
		return 515
	case png.FormatError:
		return 516
	case png.UnsupportedError:
		return 517
	case suffixarray.Index:
		return 518
	case io.LimitedReader:
		return 519
	case io.PipeReader:
		return 520
	case io.PipeWriter:
		return 521
	case io.SectionReader:
		return 522
	case log.Logger:
		return 523
	case syslog.Priority:
		return 524
	case syslog.Writer:
		return 525
	case big.Accuracy:
		return 526
	case big.ErrNaN:
		return 527
	case big.Float:
		return 528
	case big.Int:
		return 529
	case big.Rat:
		return 530
	case big.RoundingMode:
		return 531
	case big.Word:
		return 532
	case rand.Rand:
		return 533
	case rand.Zipf:
		return 534
	case mime.WordDecoder:
		return 535
	case mime.WordEncoder:
		return 536
	case multipart.FileHeader:
		return 537
	case multipart.Form:
		return 538
	case multipart.Part:
		return 539
	case multipart.Reader:
		return 540
	case multipart.Writer:
		return 541
	case quotedprintable.Reader:
		return 542
	case quotedprintable.Writer:
		return 543
	case net.AddrError:
		return 544
	case net.Buffers:
		return 545
	case net.DNSConfigError:
		return 546
	case net.DNSError:
		return 547
	case net.Dialer:
		return 548
	case net.Flags:
		return 549
	case net.HardwareAddr:
		return 550
	case net.IP:
		return 551
	case net.IPAddr:
		return 552
	case net.IPConn:
		return 553
	case net.IPMask:
		return 554
	case net.IPNet:
		return 555
	case net.Interface:
		return 556
	case net.InvalidAddrError:
		return 557
	case net.ListenConfig:
		return 558
	case net.MX:
		return 559
	case net.NS:
		return 560
	case net.OpError:
		return 561
	case net.ParseError:
		return 562
	case net.Resolver:
		return 563
	case net.SRV:
		return 564
	case net.TCPAddr:
		return 565
	case net.TCPConn:
		return 566
	case net.TCPListener:
		return 567
	case net.UDPAddr:
		return 568
	case net.UDPConn:
		return 569
	case net.UnixAddr:
		return 570
	case net.UnixConn:
		return 571
	case net.UnixListener:
		return 572
	case net.UnknownNetworkError:
		return 573
	case http.Client:
		return 574
	case http.ConnState:
		return 575
	case http.Cookie:
		return 576
	case http.Dir:
		return 577
	case http.HandlerFunc:
		return 578
	case http.Header:
		return 579
	case http.ProtocolError:
		return 580
	case http.PushOptions:
		return 581
	case http.Request:
		return 582
	case http.Response:
		return 583
	case http.SameSite:
		return 584
	case http.ServeMux:
		return 585
	case http.Server:
		return 586
	case http.Transport:
		return 587
	case cgi.Handler:
		return 588
	case cookiejar.Jar:
		return 589
	case cookiejar.Options:
		return 590
	case httptest.ResponseRecorder:
		return 591
	case httptest.Server:
		return 592
	case httptrace.ClientTrace:
		return 593
	case httptrace.DNSDoneInfo:
		return 594
	case httptrace.DNSStartInfo:
		return 595
	case httptrace.GotConnInfo:
		return 596
	case httptrace.WroteRequestInfo:
		return 597
	case httputil.ClientConn:
		return 598
	case httputil.ReverseProxy:
		return 599
	case httputil.ServerConn:
		return 600
	case mail.Address:
		return 601
	case mail.AddressParser:
		return 602
	case mail.Header:
		return 603
	case mail.Message:
		return 604
	case rpc.Call:
		return 605
	case rpc.Client:
		return 606
	case rpc.Request:
		return 607
	case rpc.Response:
		return 608
	case rpc.Server:
		return 609
	case rpc.ServerError:
		return 610
	case smtp.Client:
		return 611
	case smtp.ServerInfo:
		return 612
	case textproto.Conn:
		return 613
	case textproto.Error:
		return 614
	case textproto.MIMEHeader:
		return 615
	case textproto.Pipeline:
		return 616
	case textproto.ProtocolError:
		return 617
	case textproto.Reader:
		return 618
	case textproto.Writer:
		return 619
	case url.Error:
		return 620
	case url.EscapeError:
		return 621
	case url.InvalidHostError:
		return 622
	case url.URL:
		return 623
	case url.Userinfo:
		return 624
	case url.Values:
		return 625
	case os.File:
		return 626
	case os.FileMode:
		return 627
	case os.LinkError:
		return 628
	case os.PathError:
		return 629
	case os.ProcAttr:
		return 630
	case os.Process:
		return 631
	case os.ProcessState:
		return 632
	case os.SyscallError:
		return 633
	case exec.Cmd:
		return 634
	case exec.Error:
		return 635
	case exec.ExitError:
		return 636
	case user.Group:
		return 637
	case user.UnknownGroupError:
		return 638
	case user.UnknownGroupIdError:
		return 639
	case user.UnknownUserError:
		return 640
	case user.UnknownUserIdError:
		return 641
	case user.User:
		return 642
	case filepath.WalkFunc:
		return 643
	case plugin.Plugin:
		return 644
	case reflect.ChanDir:
		return 645
	case reflect.Kind:
		return 646
	case reflect.MapIter:
		return 647
	case reflect.Method:
		return 648
	case reflect.SelectCase:
		return 649
	case reflect.SelectDir:
		return 650
	case reflect.SliceHeader:
		return 651
	case reflect.StringHeader:
		return 652
	case reflect.StructField:
		return 653
	case reflect.StructTag:
		return 654
	case reflect.Value:
		return 655
	case reflect.ValueError:
		return 656
	case regexp.Regexp:
		return 657
	case syntax.EmptyOp:
		return 658
	case syntax.Error:
		return 659
	case syntax.ErrorCode:
		return 660
	case syntax.Flags:
		return 661
	case syntax.Inst:
		return 662
	case syntax.InstOp:
		return 663
	case syntax.Op:
		return 664
	case syntax.Prog:
		return 665
	case syntax.Regexp:
		return 666
	case runtime.BlockProfileRecord:
		return 667
	case runtime.Frame:
		return 668
	case runtime.Frames:
		return 669
	case runtime.Func:
		return 670
	case runtime.MemProfileRecord:
		return 671
	case runtime.MemStats:
		return 672
	case runtime.StackRecord:
		return 673
	case runtime.TypeAssertionError:
		return 674
	case debug.BuildInfo:
		return 675
	case debug.GCStats:
		return 676
	case debug.Module:
		return 677
	case pprof.LabelSet:
		return 678
	case pprof.Profile:
		return 679
	case trace.Region:
		return 680
	case trace.Task:
		return 681
	case sort.Float64Slice:
		return 682
	case sort.IntSlice:
		return 683
	case sort.StringSlice:
		return 684
	case strconv.NumError:
		return 685
	case strings.Builder:
		return 686
	case strings.Reader:
		return 687
	case strings.Replacer:
		return 688
	case sync.Cond:
		return 689
	case sync.Map:
		return 690
	case sync.Mutex:
		return 691
	case sync.Once:
		return 692
	case sync.Pool:
		return 693
	case sync.RWMutex:
		return 694
	case sync.WaitGroup:
		return 695
	case atomic.Value:
		return 696
	case syscall.Cmsghdr:
		return 697
	case syscall.Credential:
		return 698
	case syscall.Dirent:
		return 699
	case syscall.EpollEvent:
		return 700
	case syscall.Errno:
		return 701
	case syscall.FdSet:
		return 702
	case syscall.Flock_t:
		return 703
	case syscall.Fsid:
		return 704
	case syscall.ICMPv6Filter:
		return 705
	case syscall.IPMreq:
		return 706
	case syscall.IPMreqn:
		return 707
	case syscall.IPv6MTUInfo:
		return 708
	case syscall.IPv6Mreq:
		return 709
	case syscall.IfAddrmsg:
		return 710
	case syscall.IfInfomsg:
		return 711
	case syscall.Inet4Pktinfo:
		return 712
	case syscall.Inet6Pktinfo:
		return 713
	case syscall.InotifyEvent:
		return 714
	case syscall.Iovec:
		return 715
	case syscall.Linger:
		return 716
	case syscall.Msghdr:
		return 717
	case syscall.NetlinkMessage:
		return 718
	case syscall.NetlinkRouteAttr:
		return 719
	case syscall.NetlinkRouteRequest:
		return 720
	case syscall.NlAttr:
		return 721
	case syscall.NlMsgerr:
		return 722
	case syscall.NlMsghdr:
		return 723
	case syscall.ProcAttr:
		return 724
	case syscall.PtraceRegs:
		return 725
	case syscall.RawSockaddr:
		return 726
	case syscall.RawSockaddrAny:
		return 727
	case syscall.RawSockaddrInet4:
		return 728
	case syscall.RawSockaddrInet6:
		return 729
	case syscall.RawSockaddrLinklayer:
		return 730
	case syscall.RawSockaddrNetlink:
		return 731
	case syscall.RawSockaddrUnix:
		return 732
	case syscall.Rlimit:
		return 733
	case syscall.RtAttr:
		return 734
	case syscall.RtGenmsg:
		return 735
	case syscall.RtMsg:
		return 736
	case syscall.RtNexthop:
		return 737
	case syscall.Rusage:
		return 738
	case syscall.Signal:
		return 739
	case syscall.SockFilter:
		return 740
	case syscall.SockFprog:
		return 741
	case syscall.SockaddrInet4:
		return 742
	case syscall.SockaddrInet6:
		return 743
	case syscall.SockaddrLinklayer:
		return 744
	case syscall.SockaddrNetlink:
		return 745
	case syscall.SockaddrUnix:
		return 746
	case syscall.SocketControlMessage:
		return 747
	case syscall.Stat_t:
		return 748
	case syscall.Statfs_t:
		return 749
	case syscall.SysProcAttr:
		return 750
	case syscall.SysProcIDMap:
		return 751
	case syscall.Sysinfo_t:
		return 752
	case syscall.TCPInfo:
		return 753
	case syscall.Termios:
		return 754
	case syscall.Time_t:
		return 755
	case syscall.Timespec:
		return 756
	case syscall.Timeval:
		return 757
	case syscall.Timex:
		return 758
	case syscall.Tms:
		return 759
	case syscall.Ucred:
		return 760
	case syscall.Ustat_t:
		return 761
	case syscall.Utimbuf:
		return 762
	case syscall.Utsname:
		return 763
	case syscall.WaitStatus:
		return 764
	case testing.B:
		return 765
	case testing.BenchmarkResult:
		return 766
	case testing.Cover:
		return 767
	case testing.CoverBlock:
		return 768
	case testing.InternalBenchmark:
		return 769
	case testing.InternalExample:
		return 770
	case testing.InternalTest:
		return 771
	case testing.M:
		return 772
	case testing.PB:
		return 773
	case testing.T:
		return 774
	case quick.CheckEqualError:
		return 775
	case quick.CheckError:
		return 776
	case quick.Config:
		return 777
	case quick.SetupError:
		return 778
	case text_scanner.Position:
		return 779
	case text_scanner.Scanner:
		return 780
	case tabwriter.Writer:
		return 781
	case text_template.ExecError:
		return 782
	case text_template.FuncMap:
		return 783
	case text_template.Template:
		return 784
	case parse.ActionNode:
		return 785
	case parse.BoolNode:
		return 786
	case parse.BranchNode:
		return 787
	case parse.ChainNode:
		return 788
	case parse.CommandNode:
		return 789
	case parse.DotNode:
		return 790
	case parse.FieldNode:
		return 791
	case parse.IdentifierNode:
		return 792
	case parse.IfNode:
		return 793
	case parse.ListNode:
		return 794
	case parse.NilNode:
		return 795
	case parse.NodeType:
		return 796
	case parse.NumberNode:
		return 797
	case parse.PipeNode:
		return 798
	case parse.Pos:
		return 799
	case parse.RangeNode:
		return 800
	case parse.StringNode:
		return 801
	case parse.TemplateNode:
		return 802
	case parse.TextNode:
		return 803
	case parse.Tree:
		return 804
	case parse.VariableNode:
		return 805
	case parse.WithNode:
		return 806
	case time.Duration:
		return 807
	case time.Location:
		return 808
	case time.Month:
		return 809
	case time.ParseError:
		return 810
	case time.Ticker:
		return 811
	case time.Time:
		return 812
	case time.Timer:
		return 813
	case time.Weekday:
		return 814
	case unicode.CaseRange:
		return 815
	case unicode.Range16:
		return 816
	case unicode.Range32:
		return 817
	case unicode.RangeTable:
		return 818
	case unicode.SpecialCase:
		return 819
	case unsafe.Pointer:
		return 820
	case reflect.Type:  // Specificity=31
		return 821
	case testing.TB:  // Specificity=18
		return 822
	case types.Object:  // Specificity=16
		return 823
	case net.Conn:  // Specificity=8
		return 824
	case binary.ByteOrder:  // Specificity=7
		return 825
	case net.PacketConn:  // Specificity=7
		return 826
	case elliptic.Curve:  // Specificity=6
		return 827
	case fmt.ScanState:  // Specificity=6
		return 828
	case hash.Hash32:  // Specificity=6
		return 829
	case hash.Hash64:  // Specificity=6
		return 830
	case os.FileInfo:  // Specificity=6
		return 831
	case parse.Node:  // Specificity=6
		return 832
	case heap.Interface:  // Specificity=5
		return 833
	case driver.RowsNextResultSet:  // Specificity=5
		return 834
	case hash.Hash:  // Specificity=5
		return 835
	case http.File:  // Specificity=5
		return 836
	case context.Context:  // Specificity=4
		return 837
	case cipher.AEAD:  // Specificity=4
		return 838
	case driver.RowsColumnTypeDatabaseTypeName:  // Specificity=4
		return 839
	case driver.RowsColumnTypeLength:  // Specificity=4
		return 840
	case driver.RowsColumnTypeNullable:  // Specificity=4
		return 841
	case driver.RowsColumnTypePrecisionScale:  // Specificity=4
		return 842
	case driver.RowsColumnTypeScanType:  // Specificity=4
		return 843
	case driver.Stmt:  // Specificity=4
		return 844
	case fmt.State:  // Specificity=4
		return 845
	case constant.Value:  // Specificity=4
		return 846
	case image.PalettedImage:  // Specificity=4
		return 847
	case draw.Image:  // Specificity=4
		return 848
	case multipart.File:  // Specificity=4
		return 849
	case rpc.ClientCodec:  // Specificity=4
		return 850
	case rpc.ServerCodec:  // Specificity=4
		return 851
	case cipher.Block:  // Specificity=3
		return 852
	case driver.Conn:  // Specificity=3
		return 853
	case driver.Rows:  // Specificity=3
		return 854
	case dwarf.Type:  // Specificity=3
		return 855
	case flag.Getter:  // Specificity=3
		return 856
	case ast.Decl:  // Specificity=3
		return 857
	case ast.Expr:  // Specificity=3
		return 858
	case ast.Spec:  // Specificity=3
		return 859
	case ast.Stmt:  // Specificity=3
		return 860
	case types.Sizes:  // Specificity=3
		return 861
	case image.Image:  // Specificity=3
		return 862
	case io.ReadWriteCloser:  // Specificity=3
		return 863
	case io.ReadWriteSeeker:  // Specificity=3
		return 864
	case rand.Source64:  // Specificity=3
		return 865
	case net.Error:  // Specificity=3
		return 866
	case net.Listener:  // Specificity=3
		return 867
	case http.ResponseWriter:  // Specificity=3
		return 868
	case sort.Interface:  // Specificity=3
		return 869
	case syscall.RawConn:  // Specificity=3
		return 870
	case flate.Reader:  // Specificity=2
		return 871
	case crypto.Decrypter:  // Specificity=2
		return 872
	case crypto.Signer:  // Specificity=2
		return 873
	case cipher.BlockMode:  // Specificity=2
		return 874
	case tls.ClientSessionCache:  // Specificity=2
		return 875
	case sql.Result:  // Specificity=2
		return 876
	case driver.Connector:  // Specificity=2
		return 877
	case driver.Result:  // Specificity=2
		return 878
	case driver.Tx:  // Specificity=2
		return 879
	case flag.Value:  // Specificity=2
		return 880
	case ast.Node:  // Specificity=2
		return 881
	case types.ImporterFrom:  // Specificity=2
		return 882
	case types.Type:  // Specificity=2
		return 883
	case jpeg.Reader:  // Specificity=2
		return 884
	case png.EncoderBufferPool:  // Specificity=2
		return 885
	case io.ByteScanner:  // Specificity=2
		return 886
	case io.ReadCloser:  // Specificity=2
		return 887
	case io.ReadSeeker:  // Specificity=2
		return 888
	case io.ReadWriter:  // Specificity=2
		return 889
	case io.RuneScanner:  // Specificity=2
		return 890
	case io.WriteCloser:  // Specificity=2
		return 891
	case io.WriteSeeker:  // Specificity=2
		return 892
	case rand.Source:  // Specificity=2
		return 893
	case net.Addr:  // Specificity=2
		return 894
	case http.CookieJar:  // Specificity=2
		return 895
	case cookiejar.PublicSuffixList:  // Specificity=2
		return 896
	case httputil.BufferPool:  // Specificity=2
		return 897
	case smtp.Auth:  // Specificity=2
		return 898
	case os.Signal:  // Specificity=2
		return 899
	case runtime.Error:  // Specificity=2
		return 900
	case sync.Locker:  // Specificity=2
		return 901
	case flate.Resetter:  // Specificity=1
		return 902
	case zlib.Resetter:  // Specificity=1
		return 903
	case crypto.SignerOpts:  // Specificity=1
		return 904
	case cipher.Stream:  // Specificity=1
		return 905
	case sql.Scanner:  // Specificity=1
		return 906
	case driver.ColumnConverter:  // Specificity=1
		return 907
	case driver.ConnBeginTx:  // Specificity=1
		return 908
	case driver.ConnPrepareContext:  // Specificity=1
		return 909
	case driver.Driver:  // Specificity=1
		return 910
	case driver.DriverContext:  // Specificity=1
		return 911
	case driver.Execer:  // Specificity=1
		return 912
	case driver.ExecerContext:  // Specificity=1
		return 913
	case driver.NamedValueChecker:  // Specificity=1
		return 914
	case driver.Pinger:  // Specificity=1
		return 915
	case driver.Queryer:  // Specificity=1
		return 916
	case driver.QueryerContext:  // Specificity=1
		return 917
	case driver.SessionResetter:  // Specificity=1
		return 918
	case driver.StmtExecContext:  // Specificity=1
		return 919
	case driver.StmtQueryContext:  // Specificity=1
		return 920
	case driver.Validator:  // Specificity=1
		return 921
	case driver.ValueConverter:  // Specificity=1
		return 922
	case driver.Valuer:  // Specificity=1
		return 923
	case macho.Load:  // Specificity=1
		return 924
	case encoding.BinaryMarshaler:  // Specificity=1
		return 925
	case encoding.BinaryUnmarshaler:  // Specificity=1
		return 926
	case encoding.TextMarshaler:  // Specificity=1
		return 927
	case encoding.TextUnmarshaler:  // Specificity=1
		return 928
	case gob.GobDecoder:  // Specificity=1
		return 929
	case gob.GobEncoder:  // Specificity=1
		return 930
	case json.Marshaler:  // Specificity=1
		return 931
	case json.Unmarshaler:  // Specificity=1
		return 932
	case xml.Marshaler:  // Specificity=1
		return 933
	case xml.MarshalerAttr:  // Specificity=1
		return 934
	case xml.TokenReader:  // Specificity=1
		return 935
	case xml.Unmarshaler:  // Specificity=1
		return 936
	case xml.UnmarshalerAttr:  // Specificity=1
		return 937
	case expvar.Var:  // Specificity=1
		return 938
	case fmt.Formatter:  // Specificity=1
		return 939
	case fmt.GoStringer:  // Specificity=1
		return 940
	case fmt.Scanner:  // Specificity=1
		return 941
	case fmt.Stringer:  // Specificity=1
		return 942
	case ast.Visitor:  // Specificity=1
		return 943
	case types.Importer:  // Specificity=1
		return 944
	case color.Color:  // Specificity=1
		return 945
	case color.Model:  // Specificity=1
		return 946
	case draw.Drawer:  // Specificity=1
		return 947
	case draw.Quantizer:  // Specificity=1
		return 948
	case io.ByteReader:  // Specificity=1
		return 949
	case io.ByteWriter:  // Specificity=1
		return 950
	case io.Closer:  // Specificity=1
		return 951
	case io.Reader:  // Specificity=1
		return 952
	case io.ReaderAt:  // Specificity=1
		return 953
	case io.ReaderFrom:  // Specificity=1
		return 954
	case io.RuneReader:  // Specificity=1
		return 955
	case io.Seeker:  // Specificity=1
		return 956
	case io.StringWriter:  // Specificity=1
		return 957
	case io.Writer:  // Specificity=1
		return 958
	case io.WriterAt:  // Specificity=1
		return 959
	case io.WriterTo:  // Specificity=1
		return 960
	case http.CloseNotifier:  // Specificity=1
		return 961
	case http.FileSystem:  // Specificity=1
		return 962
	case http.Flusher:  // Specificity=1
		return 963
	case http.Handler:  // Specificity=1
		return 964
	case http.Hijacker:  // Specificity=1
		return 965
	case http.Pusher:  // Specificity=1
		return 966
	case http.RoundTripper:  // Specificity=1
		return 967
	case syscall.Conn:  // Specificity=1
		return 968
	case syscall.Sockaddr:  // Specificity=1
		return 969
	case quick.Generator:  // Specificity=1
		return 970
	}
	return -1
}
