package x

// Data type (built-in) constants.
const (
	Void    uint8 = 0
	Int8    uint8 = 1
	Int16   uint8 = 2
	Int32   uint8 = 3
	Int64   uint8 = 4
	UInt8   uint8 = 5
	UInt16  uint8 = 6
	UInt32  uint8 = 7
	UInt64  uint8 = 8
	Boolean uint8 = 9
	String  uint8 = 10
	Float32 uint8 = 11
	Float64 uint8 = 12
)

// TypeGreaterThan reports type one is greater than type two or not.
func TypeGreaterThan(t1, t2 uint8) bool {
	switch t1 {
	case Int16:
		return t2 == Int8
	case Int32:
		return t2 == Int8 ||
			t2 == Int16
	case Int64:
		return t2 == Int8 ||
			t2 == Int16 ||
			t2 == Int32
	case UInt16:
		return t2 == UInt8
	case UInt32:
		return t2 == UInt8 ||
			t2 == UInt16
	case UInt64:
		return t2 == UInt8 ||
			t2 == UInt16 ||
			t2 == UInt32
	}
	return false
}

// TypeAreCompatible reports type one and type two is compatible or not.
func TypesAreCompatible(t1, t2 uint8) bool {
	switch t1 {
	case Int8:
		return t2 == Int8 ||
			t2 == Int16 ||
			t2 == Int32 ||
			t2 == Int64
	case Int16:
		return t2 == Int16 ||
			t2 == Int32 ||
			t2 == Int64
	case Int32:
		return t2 == Int32 ||
			t2 == Int64
	case Int64:
		return t2 == Int64
	case UInt8:
		return t2 == UInt8 ||
			t2 == UInt16 ||
			t2 == UInt32 ||
			t2 == UInt64
	case UInt16:
		return t2 == UInt16 ||
			t2 == UInt32 ||
			t2 == UInt64
	case UInt32:
		return t2 == UInt32 ||
			t2 == UInt64
	case UInt64:
		return t2 == UInt64
	case Boolean:
		return t2 == Boolean
	}
	return false
}

// IsNumericType reports type is numeric or not.
func IsNumericType(t uint8) bool {
	return IsSignedNumericType(t) || IsUnsignedNumericType(t)
}

// IsSignedNumericType reports type is signed numeric or not.
func IsSignedNumericType(t uint8) bool {
	return t == Int8 ||
		t == Int16 ||
		t == Int32 ||
		t == Int64 ||
		t == Float32 ||
		t == Float64
}

// IsUnsignedNumericType reports type is unsigned numeric or not.
func IsUnsignedNumericType(t uint8) bool {
	return t == UInt8 ||
		t == UInt16 ||
		t == UInt32 ||
		t == UInt64
}

// TypeFromName returns type name of specified type code.
func TypeFromName(name string) uint8 {
	switch name {
	case "int8":
		return Int8
	case "int16":
		return Int16
	case "int32":
		return Int32
	case "int64":
		return Int64
	case "uint8":
		return UInt8
	case "uint16":
		return UInt16
	case "uint32":
		return UInt32
	case "uint64":
		return UInt64
	case "str":
		return String
	case "bool":
		return Boolean
	case "float32":
		return Float32
	case "float64":
		return Float64
	}
	return 0 // Unreachable code.
}

func CxxTypeNameFromType(typeCode uint8) string {
	switch typeCode {
	case Void:
		return "void"
	case Int8:
		return "signed char"
	case Int16:
		return "short"
	case Int32:
		return "int"
	case Int64:
		return "long"
	case UInt8:
		return "unsigned char"
	case UInt16:
		return "unsigned short"
	case UInt32:
		return "unsigned int"
	case UInt64:
		return "unsigned long"
	case Boolean:
		return "bool"
	case Float32:
		return "float"
	case Float64:
		return "double"
	}
	return "" // Unreachable code.
}
