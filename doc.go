// Package keyvalues reads and writes Valve's KeyValues text format, as used by
// gameinfo.txt, .vmt, .vmf, .res and many other Source engine files.
//
// A document is a tree of KeyValue nodes. Each node is either a leaf, which
// holds a string value, or a block, which holds child nodes:
//
//	"GameInfo"
//	{
//		"game"	"Counter-Strike Source"
//		FileSystem
//		{
//			SteamAppId	240	// comments run to the end of the line
//		}
//	}
//
// Reader.Read returns a document root: a block with an empty key whose
// children are the top-level nodes of the input. Keys are matched ignoring
// case, as the engine does.
//
// Values are stored as strings and only converted when asked, with
// KeyValue.AsInt or KeyValue.AsFloat.
//
// Conditionals such as [$WIN32] are kept on the node they follow and can be
// read with KeyValue.Condition. They are not evaluated.
//
// Directives such as #base and #include are not resolved. They are read as
// ordinary leaves at the top level of the document.
//
// Input may be UTF-8, or UTF-16 starting with a byte order mark, as
// localization files are. Writer.UseUTF16 writes UTF-16.
//
// Escape sequences such as \n and \" are off by default, because the engine
// does not process them in most files: backslashes are common in material
// paths. Use Reader.UseEscapeSequences and Writer.UseEscapeSequences for files
// that use them.
package keyvalues
