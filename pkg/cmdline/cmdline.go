package cmdline

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Cmdline struct {
	handle   handle
	capacity int64
}

func New(handle handle, capacity int64) *Cmdline {
	return &Cmdline{
		handle:   handle,
		capacity: capacity,
	}
}

func (c *Cmdline) String() (string, error) {
	reader := make([]byte, c.capacity)
	_, err := c.handle.ReadAt(reader, 0)
	if err != nil {
		return "", err
	}
	return string(reader), nil
}

func (c *Cmdline) Replace(cmdline string) error {
	if len(cmdline) > int(c.capacity) {
		return errors.New("cmdline too big for capacity")
	}
	producer := &writerAtProducer{
		WriterAt: c.handle,
	}
	if _, err := producer.Write([]byte(cmdline)); err != nil {
		return err
	}
	padSize := c.capacity - int64(len(cmdline))
	pad := padding(padSize)
	if _, err := producer.Write(pad); err != nil {
		return err
	}
	return nil
}

func (c *Cmdline) Append(extra string) error {
	// defragment the cmdline and get the slack
	slack, err := c.defragment()
	if err != nil {
		return err
	}
	// check if there is enough space
	if slack < int64(len(extra)) {
		return errors.New("not enough space")
	}
	producer := &writerAtProducer{
		WriterAt: c.handle,
		offset:   c.capacity - slack,
	}
	// append
	_, err = producer.Write([]byte(extra))
	return err
}

func (c *Cmdline) Set(pairs map[string]string, keepExisting bool) error {
	// merge existing and new pairs
	var merged map[string]string
	if keepExisting {
		existing, err := c.getKeyValuePairs()
		if err != nil {
			return err
		}
		merged = existing
	} else {
		merged = make(map[string]string)
	}
	for key, value := range pairs {
		merged[key] = value
	}

	// calculate required size
	var requiredSize int64
	for key, value := range merged {
		requiredSize += int64(len(key) + len(value))
		if len(value) > 0 {
			requiredSize++ // for '='
		}
	}
	requiredSize += max(int64(len(merged))-1, 0) // for spaces
	if requiredSize > c.capacity {
		return errors.New("not enough space")
	}

	// write cmdline
	sortedKeys := make([]string, 0, len(merged))
	for key := range merged {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)
	producer := &writerAtProducer{
		WriterAt: c.handle,
	}
	for i, key := range sortedKeys {
		if i > 0 {
			producer.Write([]byte{' '})
		}
		if _, err := producer.Write([]byte(key)); err != nil {
			return err
		}
		if len(merged[key]) > 0 {
			if _, err := producer.Write([]byte{'='}); err != nil {
				return err
			}
			if _, err := producer.Write([]byte(merged[key])); err != nil {
				return err
			}
		}
	}

	// pad with spaces
	padSize := c.capacity - requiredSize
	pad := padding(padSize)
	if _, err := producer.Write(pad); err != nil {
		return err
	}
	return nil
}

func (c *Cmdline) SetOne(key, value string, inPlace bool) error {
	if inPlace {
		return c.setInPlace([]byte(key), []byte(value))
	}
	return c.Set(map[string]string{string(key): string(value)}, true)
}

func (c *Cmdline) setInPlace(key, value []byte) error {
	sizeOfMatch := len(key) + 1 + len(value)
	if len(value) == 0 {
		sizeOfMatch--
	}
	if sizeOfMatch > int(c.capacity) {
		return errors.New("key and value too big for capacity of cmdline")
	}
	consumer := &readerAtConsumer{
		ReaderAt: c.handle,
	}
	scanner := bufio.NewScanner(consumer)
	splitterFn, offset := splitterForKeyValuePair(key, value)
	scanner.Split(splitterFn)
	if !scanner.Scan() {
		return fmt.Errorf("key %q not found", key)
	}
	pair := scanner.Bytes()
	if sizeOfMatch > len(pair) {
		return errors.New("key and value too big to replace in place")
	}
	padSize := len(pair) - sizeOfMatch
	pad := padding(int64(padSize))
	producer := &writerAtProducer{
		WriterAt: c.handle,
		offset:   int64(*offset),
	}
	producer.Write(key)
	if len(value) > 0 {
		producer.Write([]byte{'='})
		producer.Write(value)
	}
	producer.Write(pad)
	return nil
}

func (c *Cmdline) getKeyValuePairs() (map[string]string, error) {
	consumer := &readerAtConsumer{
		ReaderAt: c.handle,
	}
	scanner := bufio.NewScanner(consumer)
	scanner.Split(cmdlineSplitFunc)
	pairs := make(map[string]string)
	for scanner.Scan() {
		pair := scanner.Text()
		splitPair := strings.SplitN(pair, "=", 2)
		if len(splitPair) == 1 {
			pairs[splitPair[0]] = ""
		} else if len(splitPair) == 2 {
			pairs[splitPair[0]] = splitPair[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return pairs, nil
}

// defragment removes all spaces from the cmdline and pads it with spaces
// to the capacity of the cmdline
// returns the number of bytes that were padded (usable for appending)
func (c *Cmdline) defragment() (int64, error) {
	consumer := &readerAtConsumer{
		ReaderAt: c.handle,
	}
	producer := &writerAtProducer{
		WriterAt: c.handle,
	}
	scanner := bufio.NewScanner(consumer)
	scanner.Split(cmdlineSplitFunc)

	if !scanner.Scan() {
		return c.capacity, nil // empty cmdline
	}
	// first scan is not prefixed with a space
	if _, err := producer.Write([]byte(scanner.Text())); err != nil {
		return 0, err
	}

	// all other scans are prefixed with a space
	for scanner.Scan() {
		if _, err := producer.Write([]byte{' '}); err != nil {
			return 0, err
		}
		if _, err := producer.Write([]byte(scanner.Text())); err != nil {
			return 0, err
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	// pad with spaces
	padSize := c.capacity - producer.offset
	pad := padding(padSize)
	_, err := producer.Write(pad)
	return max(padSize-1, 0), err
}

type handle interface {
	io.ReaderAt
	io.WriterAt
}

type sectionHandle struct {
	handle       handle
	offset, size int64
}

func NewSectionHandle(handle handle, offset, size int64) handle {
	return sectionHandle{
		handle: handle,
		offset: offset,
		size:   size,
	}
}

func (s sectionHandle) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.size {
		return 0, io.EOF
	}
	// can do full read?
	canRead := min(int64(len(p)), s.size-off)
	n, err = s.handle.ReadAt(p[:canRead], s.offset+off)
	if err != nil {
		return n, err
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (s sectionHandle) WriteAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.size {
		return 0, io.EOF
	}
	// can do full write?
	canWrite := min(int64(len(p)), s.size-off)
	n, err = s.handle.WriteAt(p[:canWrite], s.offset+off)
	if err != nil {
		return n, err
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

type readerAtConsumer struct {
	io.ReaderAt
	offset int64
}

func (r *readerAtConsumer) Read(p []byte) (int, error) {
	n, err := r.ReaderAt.ReadAt(p, r.offset)
	r.offset += int64(n)
	return n, err
}

type writerAtProducer struct {
	io.WriterAt
	offset int64
}

func (w *writerAtProducer) Write(p []byte) (int, error) {
	n, err := w.WriterAt.WriteAt(p, w.offset)
	w.offset += int64(n)
	return n, err
}

func cmdlineSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Return nothing if at end of file and no data passed
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	var start int
	var foundToken bool
	// Skip leading spaces
	for i := 0; i < len(data); i++ {
		if data[i] != ' ' {
			start = i
			foundToken = true
			break
		}
	}
	if !foundToken {
		return len(data), nil, nil
	}

	// Scan until space, marking end of pair
	for i := start; i < len(data); i++ {
		if data[i] == ' ' {
			return i + 1, data[start:i], nil
		}
	}

	if atEOF {
		return len(data), data[start:], nil
	}

	// If we're not at EOF, request more data
	return 0, nil, nil
}

func splitterForKeyValuePair(key, value []byte) (func(data []byte, atEOF bool) (advance int, token []byte, err error), *int) {
	var needle []byte
	var offset int
	if len(value) == 0 {
		needle = make([]byte, len(key)+1)
		copy(needle, key)
		needle[len(key)] = ' '
	} else {
		needle = make([]byte, len(key)+1)
		copy(needle, key)
		needle[len(key)] = '='
	}

	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		// Return nothing if at end of file and no data passed
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		keyIndex := strings.Index(string(data), string(needle))
		if keyIndex == -1 {
			offset += len(data)
			return len(data), nil, nil
		}

		// ensure that the key is not a substring of a longer key
		if keyIndex > 0 && data[keyIndex-1] != ' ' {
			offset += len(data)
			return len(data), nil, nil
		}

		// Scan until space, marking end of pair
		for i := keyIndex; i < len(data); i++ {
			if data[i] == ' ' {
				offset += keyIndex
				return i + 1, data[keyIndex:i], nil
			}
		}

		if atEOF {
			offset += keyIndex
			return len(data), data[keyIndex:], nil
		}

		// If we're not at EOF, request more data
		return 0, nil, nil
	}, &offset
}

func padding(size int64) []byte {
	pad := make([]byte, size)
	for i := range pad {
		pad[i] = ' '
	}
	return pad
}
