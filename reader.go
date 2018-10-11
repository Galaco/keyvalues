package keyvalues

import (
	"strings"
	"bufio"
	"io"
)

const CHAR_ENTER_SCOPE = "{"
const CHAR_EXIT_SCOPE = "}"
const CHAR_ESCAPE = "\""
const CHAR_DISCARD_CUTSET = "\t \r\n"
const CHAR_SEPARATOR = " "
const CHAR_TAB = "\t"
const CHAR_COMMENT = "//"
const NODE_KEY_ROOT = "$root"

type Reader struct {
	file io.Reader
}

// Return a new Vmf Reader
func NewReader(file io.Reader) Reader {
	reader := Reader{}
	reader.file = file
	return reader
}

// Read buffer file into our defined structures
// Returns a fully mapped Vmf structure
func (reader *Reader) Read() (keyvalue KeyValue, err error) {
	bufReader := bufio.NewReader(reader.file)

	rootNode := KeyValue{
		key: NODE_KEY_ROOT,
	}

	readScope(bufReader, &rootNode)

	return rootNode,err
}

// Read a single scope
// Constructs a KeyValue node tree for a single scope
// Recursively parses all child scopes too
// Param: scope is the current scope to write to
func readScope(reader *bufio.Reader, scope *KeyValue) *KeyValue {
	for {
		line,err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}

		// Remove any comments
		line = strings.Split(line, CHAR_COMMENT)[0]
		// trim padding
		line = strings.Trim(line, CHAR_DISCARD_CUTSET)
		// Simplify parsing the line
		line = strings.Replace(line, CHAR_TAB, CHAR_SEPARATOR, -1)

		if len(line) == 0 {
			continue
		}


		// New scopez
		if strings.Contains(line, CHAR_ENTER_SCOPE) {
			// Scope is opened when the key is read
			// There may be situations where there is no key, so we must account for that
			subScope := scope.value[len(scope.value) - 1].(KeyValue)
			scope.value = append(scope.value[:len(scope.value)-1], *readScope(reader, &subScope))
			continue
		}

		// Exit scope
		if strings.Contains(line, CHAR_EXIT_SCOPE) {
			break
		}

		// Read scope
		prop := strings.Split(line, CHAR_SEPARATOR)

		// Only the key is defined here
		// This *SHOULD* mean key has children
		if len(prop) == 1 {
			//Create new scope
			kv := KeyValue{
				key: strings.Replace(prop[0], CHAR_ESCAPE, "", -1),
				valueType: ValueArray,
			}

			scope.value = append(scope.value, kv)
			continue
		}


		// Read keyvalue & append to current scope
		scope.value = append(scope.value, parseKV(line))
	}

	return scope
}

func parseKV(line string) KeyValue {
	prop := strings.Split(line, CHAR_SEPARATOR)
	// value also defined on this line
	val := strings.Replace(strings.Replace(line, prop[0] + CHAR_SEPARATOR, "", -1), CHAR_ESCAPE, "", -1)

	return KeyValue{
		key: prop[0],
		valueType: getType(val),
		value: append(make([]interface{}, 0), val),
	}
}