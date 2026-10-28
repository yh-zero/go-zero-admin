package orm

import "strings"

// redactSQL handles raw SQL literals as well as parameterized statements.
func redactSQL(sql string) string {
	var output strings.Builder
	for i := 0; i < len(sql); {
		switch {
		case sql[i] == '\'' || sql[i] == '"':
			quote := sql[i]
			i++
			for i < len(sql) {
				if sql[i] == '\\' {
					i += 2
					continue
				}
				if sql[i] == quote {
					i++
					if i < len(sql) && sql[i] == quote {
						i++
						continue
					}
					break
				}
				i++
			}
			output.WriteByte('?')
		case sql[i] == '`':
			start := i
			i++
			for i < len(sql) {
				if sql[i] == '`' {
					i++
					if i < len(sql) && sql[i] == '`' {
						i++
						continue
					}
					break
				}
				i++
			}
			output.WriteString(sql[start:i])
		case sql[i] == '#' || (i+1 < len(sql) && sql[i:i+2] == "--"):
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			output.WriteByte(' ')
		case i+1 < len(sql) && sql[i:i+2] == "/*":
			i += 2
			for i+1 < len(sql) && sql[i:i+2] != "*/" {
				i++
			}
			i = min(i+2, len(sql))
			output.WriteByte(' ')
		case sql[i] >= '0' && sql[i] <= '9' && (i == 0 || !sqlIdentifierByte(sql[i-1])):
			i++
			for i < len(sql) && (sqlIdentifierByte(sql[i]) || sql[i] == '.' || sql[i] == '+' || sql[i] == '-') {
				i++
			}
			output.WriteByte('?')
		default:
			output.WriteByte(sql[i])
			i++
		}
	}
	return output.String()
}

func sqlIdentifierByte(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_' || char >= 128
}
