// package rotateloghook is a port of File-RotateLog from Perl
// (https://metacpan.org/release/File-RotateLog), and it allows
// you to automatically rotate output files when you write to them
// according to the filename pattern that you can specify.
package rotateloghook

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	strftime "github.com/lestrrat-go/strftime"
	"github.com/pkg/errors"
)

func (c clockFn) Now() time.Time {
	return c()
}

// New creates a new RotateLog object. A log filename pattern
// must be passed. Optional `Option` parameters may be passed
func New(p string, options ...Option) (*RotateLog, error) {
	globPattern := p
	for _, re := range patternConversionRegexps {
		globPattern = re.ReplaceAllString(globPattern, "*")
	}

	pattern, err := strftime.New(p)
	if err != nil {
		return nil, errors.Wrap(err, `invalid strftime pattern`)
	}

	var clock Clock = Local
	rotationTime := 24 * time.Hour
	var rotationCount int
	var maxAge time.Duration

	res := &RotateLog{
		clock:            clock,
		globPattern:      globPattern,
		maxAge:           maxAge,
		pattern:          pattern,
		rotationTime:     rotationTime,
		rotationCount:    rotationCount,
		rotationNotifier: make(chan RotationEvent),
		rotationSize:     -1,
	}

	for _, o := range options {
		switch o.Name() {
		case OptKeyClock:
			res.clock = o.Value().(Clock)
		case OptKeyLinkName:
			res.linkName = o.Value().(string)
		case OptKeyMaxAge:
			maxAge = o.Value().(time.Duration)
			if maxAge < 0 {
				maxAge = 0
			}
			res.maxAge = maxAge
			res.RegisterPurgeChecker(res.buildFileAgeChecker())
		case OptKeyRotationCount:
			res.rotationCount = o.Value().(int)
			res.RegisterPurgeChecker(res.buildFileCountChecker())

		case OptKeyRotationTime:
			rotationTime = o.Value().(time.Duration)
			if rotationTime < 0 {
				rotationTime = 0
			}
			res.rotationTime = rotationTime
			res.RegisterRotateCondition(res.buildRotationTimeCondition())

		case OptKeyRotationSize:
			res.rotationSize = o.Value().(int64)
			res.RegisterRotateCondition(res.buildSizeCondition())
		}

	}

	if maxAge > 0 && rotationCount > 0 {
		return nil, errors.New("options MaxAge and RotationCount cannot be both set")
	}

	if maxAge == 0 && rotationCount == 0 {
		// if both are 0, give maxAge a sane default
		maxAge = 7 * 24 * time.Hour
	}

	return res, nil
}

func (rl *RotateLog) buildSizeCondition() RotationPredicate {
	return func() (bool, string) {
		fi, err := os.Stat(rl.curFn)
		if errors.Is(err, os.ErrNotExist) {
			return false, ""
		}

		if err == nil {
			if rl.rotationSize > 0 && rl.rotationSize < fi.Size() {
				s, err := rl.newNameBuilder()
				if err != nil {
					return false, ""
				}
				return true, s
			}
		}
		return false, ""
	}
}

func (rl *RotateLog) buildRotationTimeCondition() RotationPredicate {
	return func() (bool, string) {
		now := rl.clock.Now()
		elapsedTime := now.Sub(rl.lastTs)
		if elapsedTime > rl.rotationTime {
			s, err := rl.newNameBuilder()
			if err != nil {
				return false, ""
			}
			return true, s
		}
		return false, ""
	}
}

func (rl *RotateLog) counterNameBuilder(expectedFn string) (string, error) {
	res := ""
	ext := path.Ext(expectedFn)
	fp := ""
	if len(ext) > 0 {
		fp = strings.TrimSuffix(expectedFn, ext)
	}
	// we try to make 1000 attempts to create a name with an incrementing counter
	// this will allow you not to overwrite existing files if they exist
	// in worst case, we assume that the user has configured the rotation settings incorrectly
	for range make([]int, 1000) {
		rl.generation++
		if len(ext) > 0 {
			//files with extension: dir/foo.log -> dir/foo_1.log
			res = fmt.Sprintf("%s_%v%s", fp, rl.generation, ext)
		} else {
			// files without extension: dir/foo -> dir/foo_1
			res = fmt.Sprintf("%s_%v", expectedFn, rl.generation)
		}

		if _, err := os.Stat(res); errors.Is(err, os.ErrNotExist) {
			return res, nil
		}
	}
	return "", fmt.Errorf("сan't create a name with a counter based on the current file %s, check the configuration", expectedFn)

}

// build new filename: by formatted timestamp or formatted timestamp and counter suffix.
// Ensures that there is no file with an identical name in the directory
func (rl *RotateLog) newNameBuilder() (string, error) {
	expectedFn := rl.genFilename()

	fileExist := true
	if _, err := os.Stat(expectedFn); errors.Is(err, os.ErrNotExist) {
		fileExist = false
	}

	if rl.curFn != expectedFn && !fileExist {
		rl.generation = 0
		return expectedFn, nil
	} else {
		res, err := rl.counterNameBuilder(expectedFn)
		if err != nil {
			return "", err
		}
		return res, nil
	}
}

func (rl *RotateLog) buildFileAgeChecker() PurgeChecker {
	return func(filesList []string, result *map[string]struct{}) error {

		for _, fp := range filesList {

			fl, err := os.Lstat(fp)
			if err != nil {
				continue
			}

			if fl.Mode()&os.ModeSymlink == os.ModeSymlink {
				continue
			}

			fileAge := rl.clock.Now().UTC().Sub(fl.ModTime().UTC())
			if fileAge > rl.maxAge {
				(*result)[fp] = struct{}{}
				continue
			}
		}
		return nil
	}
}

func (rl *RotateLog) buildFileCountChecker() PurgeChecker {
	return func(filesList []string, result *map[string]struct{}) error {
		if rl.rotationCount > 0 {
			// Only delete if we have more than rotationCount
			if rl.rotationCount >= len(filesList) {
				return nil
			}
			list := filesList[:len(filesList)-int(rl.rotationCount)]
			for _, file := range list {
				(*result)[file] = struct{}{}
			}
		}
		return nil
	}
}

func (rl *RotateLog) RegisterRotateCondition(predicate RotationPredicate) {
	rl.rotationConditions = append(rl.rotationConditions, predicate)
}

func (rl *RotateLog) RegisterPurgeChecker(checker PurgeChecker) {
	rl.purgeCheckers = append(rl.purgeCheckers, checker)
}

// method should compute rotate conditions and return correct name for new file, if rotation needed
func (rl *RotateLog) checkConditions() (bool, string) {
	for _, condition := range rl.rotationConditions {
		res, filename := condition()
		if res {
			return res, filename
		}
	}
	return false, ""
}

func (rl *RotateLog) genFilename() string {
	now := rl.clock.Now()
	res := rl.pattern.FormatString(now)
	return res
}

// Write satisfies the io.Writer interface. It writes to the
// appropriate file handle that is currently being used.
// If we have reached rotation time, the target file gets
// automatically rotated, and also purged if necessary.
func (rl *RotateLog) Write(p []byte) (n int, err error) {
	// Guard against concurrent writes
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	out, err := rl.getWriter_nolock(false)
	if err != nil {
		return 0, errors.Wrap(err, `failed to acquite target io.Writer`)
	}

	return out.Write(p)
}

func (rl *RotateLog) GetRotationNotifier() <-chan RotationEvent {
	return rl.rotationNotifier
}

// must be locked during this operation
func (rl *RotateLog) getWriter_nolock(bailOnRotateFail bool) (io.Writer, error) {

	needRotate, filename := rl.checkConditions()
	// if we got here, then we need to create a file
	if needRotate {

		fh, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, errors.Errorf("failed to open file %s: %s", rl.pattern, err)
		}

		if err := rl.rotateNolock(filename); err != nil {
			err = errors.Wrap(err, "failed to rotate")
			// Failure to rotate is a problem, but it's really not a great
			// idea to stop your application just because you couldn't rename
			// your log.
			// We only return this error when explicitly needed.
			if bailOnRotateFail {
				return nil, err
			}
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}

		rl.outFh.Close()
		rl.outFh = fh

		stashedFileName := rl.curFn
		rl.curFn = filename
		rl.lastTs = rl.clock.Now()

		select {
		case rl.rotationNotifier <- RotationEvent{PrevFileName: stashedFileName, NewFileName: rl.curFn}:
			fmt.Fprintf(os.Stderr, "%s\n", "RBC log file successsfully rotated")
		default:
			fmt.Println("RBC log file rotated, but no handler used inside")
		}

		return fh, nil
	} else {
		return rl.outFh, nil
	}
}

// CurrentFileName returns the current file name that
// the RotateLog object is writing to
func (rl *RotateLog) CurrentFileName() string {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	return rl.curFn
}

var patternConversionRegexps = []*regexp.Regexp{
	regexp.MustCompile(`%[%+A-Za-z]`),
	regexp.MustCompile(`\*+`),
}

// Rotate forcefully rotates the log files. If the generated file name
// clash because file already exists, a numeric suffix of the form
// ".1", ".2", ".3" and so forth are appended to the end of the log file
//
// Thie method can be used in conjunction with a signal handler so to
// emulate servers that generate new log files when they receive a
// SIGHUP
func (rl *RotateLog) Rotate() error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	if _, err := rl.getWriter_nolock(true); err != nil {
		return err
	}
	return nil
}

func (rl *RotateLog) rotateNolock(filename string) error {

	if rl.linkName != "" {
		tmpLinkName := filename + `_symlink`
		if err := os.Symlink(filename, tmpLinkName); err != nil {
			return errors.Wrap(err, `failed to create new symlink`)
		}

		if err := os.Rename(tmpLinkName, rl.linkName); err != nil {
			return errors.Wrap(err, `failed to rename new symlink`)
		}
	}

	matches, err := filepath.Glob(rl.globPattern)
	if err != nil {
		return err
	}

	filesToPurge := make(map[string]struct{})

	for _, purgeChecker := range rl.purgeCheckers {
		err := purgeChecker(matches, &filesToPurge)
		if err != nil {
			return err
		}
	}

	if len(filesToPurge) <= 0 {
		return nil
	}

	go func() {
		// purge files on a separate goroutine
		for k := range filesToPurge {
			if err := os.Remove(k); err != nil {
				fmt.Printf("Failed remove file %s: %s", k, err.Error())
			}
		}
	}()

	return nil
}

// Close satisfies the io.Closer interface. You must
// call this method if you performed any writes to
// the object.
func (rl *RotateLog) Close() error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if rl.outFh == nil {
		return nil
	}

	rl.outFh.Close()
	rl.outFh = nil
	return nil
}
