package typecode

type Typecode int32

// https://docs.microsoft.com/en-us/dotnet/api/system.typecode?view=net-5.0
const (
	Boolean  Typecode = 3  // A simple type representing Boolean values of true or false.
	Byte     Typecode = 6  // An integral type representing unsigned 8-bit integers with values between 0 and 255.
	Char     Typecode = 4  // An integral type representing unsigned 16-bit integers with values between 0 and 65535. The set of possible values for the Char type corresponds to the Unicode character set.
	DateTime Typecode = 16 // A type representing a date and time value.
	DBNull   Typecode = 2  // A database null (column) value.
	Decimal  Typecode = 15 // A simple type representing values ranging from 1.0 x 10 -28 to approximately 7.9 x 10 28 with 28-29 significant digits.
	Double   Typecode = 14 // A floating point type representing values ranging from approximately 5.0 x 10 -324 to 1.7 x 10 308 with a precision of 15-16 digits.
	Empty    Typecode = 0  // A null reference.
	Int16    Typecode = 7  // An integral type representing signed 16-bit integers with values between -32768 and 32767.
	Int32    Typecode = 9  // An integral type representing signed 32-bit integers with values between -2147483648 and 2147483647.
	Int64    Typecode = 11 // An integral type representing signed 64-bit integers with values between -9223372036854775808 and 9223372036854775807.
	Object   Typecode = 1  // A general type representing any reference or value type not explicitly represented by another TypeCode.
	SByte    Typecode = 5  // An integral type representing signed 8-bit integers with values between -128 and 127.
	Single   Typecode = 13 // A floating point type representing values ranging from approximately 1.5 x 10 -45 to 3.4 x 10 38 with a precision of 7 digits.
	String   Typecode = 18 // A sealed class type representing Unicode character strings.
	UInt16   Typecode = 8  // An integral type representing unsigned 16-bit integers with values between 0 and 65535.
	UInt32   Typecode = 10 // An integral type representing unsigned 32-bit integers with values between 0 and 4294967295.
	UInt64   Typecode = 12 // An integral type representing unsigned 64-bit integers with values between 0 and 18446744073709551615.

	EventPipeTypeCodeArray Typecode = 19 // https://github.com/microsoft/perfview/blob/main/src/TraceEvent/EventPipe/EventPipeFormat.md#metadata-event-encoding
)
