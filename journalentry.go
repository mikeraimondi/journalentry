package journalentry

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mikeraimondi/frontmatter"
)

const (
	entryFormat = "2006-01-02-Journal-Entry-for-Jan-2" + ".md"
	entryRegex  = `\d{4}-\d{2}-\d{2}-Journal-Entry-for-\D{3}-\d{1,2}` + ".md"
	wordRegex   = `\S+`
	ratingRegex = `^[1-5]$`
)

// Entry represents a single journal entry.
type Entry struct {
	// TODO move FM attributes to own struct
	Seconds     uint16
	LowMood     uint8
	HighMood    uint8
	AverageMood uint8
	Body        []byte    `yaml:"-"`
	Path        string    `yaml:"-"`
	ModTime     time.Time `yaml:"-"`
}

// New reads the directory named by dir and either returns an existing Entry in that directory, or creates a new one if none exist.
func New(dir string) (p *Entry, err error) {
	info, err := os.Stat(dir)
	if err != nil {
		return p, err
	}
	if !info.IsDir() {
		return p, errors.New("must be a directory")
	}
	p = &Entry{Path: dir + string(filepath.Separator) + time.Now().Format(entryFormat)}
	if _, err := os.Stat(p.Path); os.IsNotExist(err) {
		p.ModTime = time.Now()
		err = p.Save()
	} else if err == nil {
		_, err = p.Load()
	}
	return p, err
}

// Load reads the file named by p.Path and populates the Entry
func (p *Entry) Load() (modified bool, err error) {
	f, err := os.Open(p.Path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return false, err
	}
	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	modified = info.ModTime() != p.ModTime
	p.ModTime = info.ModTime()
	p.Body, err = frontmatter.Unmarshal(data, p)
	return modified, err
}

// Save writes the Entry to the file named by p.Path
func (p *Entry) Save() (err error) {
	fm, err := frontmatter.Marshal(&p)
	if err != nil {
		return err
	}
	var perm os.FileMode = 0666
	if err = ioutil.WriteFile(p.Path, append(fm, p.Body...), perm); err != nil {
		fmt.Println("Dump:")
		fmt.Println(string(fm))
		fmt.Println(string(p.Body))
	}
	return err
}

func (p *Entry) Date() (time.Time, error) {
	return time.Parse(entryFormat, filepath.Base(p.Path))
}

// Words returns the number of words in p.body
func (p *Entry) Words() [][]byte {
	return regexp.MustCompile(wordRegex).FindAll(p.Body, -1)
}

// PromptForMetadata prints questions to w and sets the values of p based on values read from reader.
func (p *Entry) PromptForMetadata(reader io.Reader, w io.Writer) (err error) {
	r := bufio.NewReader(reader)
	for prompt, setter := range p.prompts() {
		for {
			fmt.Fprint(w, prompt)
			input, err := r.ReadString('\n')
			if err != nil {
				return err
			}
			input = strings.TrimSpace(input)
			regex := regexp.MustCompile(ratingRegex)
			if regex.MatchString(input) {
				rating, err := strconv.ParseUint(input, 10, 8)
				if err != nil {
					return err
				}
				setter(uint8(rating))
				break
			} else {
				fmt.Fprintln(w, "Unrecognized input")
			}
		}
	}
	return err
}

// IsEntry returns true if path refers to a file with an Entry-like name, false otherwise.
func IsEntry(path string) bool {
	return regexp.MustCompile(entryRegex).MatchString(path)
}

func (p *Entry) setLowMood(rating uint8) {
	p.LowMood = rating
}

func (p *Entry) setHighMood(rating uint8) {
	p.HighMood = rating
}

func (p *Entry) setAvgMood(rating uint8) {
	p.AverageMood = rating
}

func (p *Entry) prompts() (pr map[string]func(uint8)) {
	pr = make(map[string]func(uint8))
	if p.HighMood == 0 {
		pr["High mood for the day? (1-5) "] = p.setHighMood
	}
	if p.LowMood == 0 {
		pr["Low mood for the day? (1-5) "] = p.setLowMood
	}
	if p.AverageMood == 0 {
		pr["Average mood for the day? (1-5) "] = p.setAvgMood
	}
	return pr
}
