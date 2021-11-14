package fsstat

import "strconv"

func parseInt(value *int, source string) error {
	var err error
	*value, err = strconv.Atoi(source)

	return err
}
