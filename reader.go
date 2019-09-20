package keyvalues

import (
	"bufio"
	"io"
	"strings"
)

const tokenEnterScope = "{"
const tokenExitScope = "}"
const tokenEscape = "\""
const tokenDiscardCutset = "\t \r\n"
const tokenSeparator = " "
const tokenTab = "\t"
const tokenComment = "//"
const tokenRootNodeKey = "$root"

// Reader is used for parsing a KeyValue format stream
// There are various KeyValue based formats (vmt, vmf, gameinfo.txt etc.)
// This should be able to parse all of them.
type Reader struct {
	file io.Reader
}

// NewReader Return a new Vmf Reader
func NewReader(file io.Reader) Reader {
	reader := Reader{}
	reader.file = file
	return reader
}

// Read buffer file into our defined structures
// Returns a fully mapped Vmf structure
// Every root KeyValue is contained in a predefined root node, due to spec lacking clarity
// about the number of valid root nodes. This assumes there can be more than 1
func (reader *Reader) Read() (keyvalue KeyValue, err error) {
	bufReader := bufio.NewReader(reader.file)

	rootNode := KeyValue{
		key:       tokenRootNodeKey,
		valueType: ValueArray,
		parent:    nil,
	}

	readScope(bufReader, &rootNode)

	if rootNode.HasChildren() && len(rootNode.value) == 1 {
		root := rootNode.value[0].(*KeyValue)
		return *root,nil
	}

	return rootNode, err
}

// readScope Reads a single scope
// Constructs a KeyValue node tree for a single scope
// Recursively parses all child scopes too
// Param: scope is the current scope to write to
func readScope(reader *bufio.Reader, scope *KeyValue) *KeyValue {
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}

		// Remove any comments
		line = strings.Split(line, tokenComment)[0]
		// trim padding
		line = strings.Trim(line, tokenDiscardCutset)
		// Simplify parsing the line
		line = strings.Replace(line, tokenTab, tokenSeparator, -1)

		if len(line) == 0 {
			continue
		}

		// New scope
		if strings.Contains(line, tokenEnterScope) && !isCharacterEscaped(line, tokenEnterScope) {
			// Scope is opened when the key is read
			// There may be situations where there is no key, so we must account for that
			subScope := scope.value[len(scope.value)-1].(*KeyValue)
			scope.value = append(scope.value[:len(scope.value)-1], readScope(reader, subScope))
			continue
		}

		// Exit scope
		if strings.Contains(line, tokenExitScope) {
			break
		}

		// Read scope
		prop := strings.Split(line, tokenSeparator)

		// Only the key is defined here
		// This *SHOULD* mean key has children
		if len(prop) == 1 {
			//Create new scope
			kv := &KeyValue{
				key:       trim(prop[0]),
				valueType: ValueArray,
				parent:    scope,
			}

			scope.value = append(scope.value, kv)
			continue
		}

		// Read keyvalue & append to current scope
		results := parseKV(line)
		for idx := range results {
			results[idx].parent = scope
			scope.value = append(scope.value, results[idx])
		}
	}

	return scope
}

// parseKV reads a single line that should contain a KeyValue pair
func parseKV(line string) (res []*KeyValue) {
	prop := strings.Split(line, tokenSeparator)
	// value also defined on this line
	vals := strings.Split(trim(strings.Replace(line, prop[0], "", -1)), "\r")

	res = append(res, &KeyValue{
		key:       trim(prop[0]),
		valueType: getType(trim(vals[0])),
		value:     append(make([]interface{}, 0), trim(vals[0])),
	})

	// Hack to catch \r carriage returns
	if len(vals) == 2 {
		prop := strings.Split(trim(vals[1]), tokenSeparator)
		val2 := trim(strings.Replace(vals[1], prop[0], "", -1))
		res = append(res, &KeyValue{
			key:       strings.Replace(trim(prop[0]), "\"", "", -1),
			valueType: getType(val2),
			value:     append(make([]interface{}, 0), val2),
		})
	}

	return res
}

func isCharacterEscaped(value string, char string) bool {
	return strings.LastIndex(value, tokenEscape) >= strings.LastIndex(value, char)
}

func trim(value string) string {
	return strings.Trim(strings.TrimSpace(value), tokenEscape)
}
