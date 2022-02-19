package main

import (
	"errors"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	i "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	m "github.com/emersion/go-message/mail"
	"golang.org/x/text/encoding/charmap"
)

type imap struct {
	Username string
	Password string
	Server   string
	Port     string
	Client   *client.Client
}

func (imap *imap) connect() error {
	c, err := client.DialTLS(imap.Server+":"+imap.Port, nil)

	if err != nil {
		return err
	}

	imap.Client = c
	return nil
}

func (imap *imap) login() error {
	return imap.Client.Login(imap.Username, imap.Password)
}

func (imap *imap) selectMailbox(mailbox string) (*i.MailboxStatus, error) {
	return imap.Client.Select(mailbox, true)
}

func (imap *imap) search(from, to time.Time) ([]uint32, error) {
	search := i.NewSearchCriteria()
	search.Since = from
	search.Before = to

	return imap.Client.UidSearch(search)
}

func (imap *imap) createSeqSet(uids []uint32) *i.SeqSet {
	seqset := new(i.SeqSet)
	seqset.AddNum(uids...)

	return seqset
}

func (imap *imap) enableCharsetReader() {
	charset.RegisterEncoding("ansi", charmap.Windows1252)
	charset.RegisterEncoding("iso8859-15", charmap.ISO8859_15)
	i.CharsetReader = charset.Reader
}

func (imap *imap) fixUtf(str string) string {
	callable := func(r rune) rune {
		if r == utf8.RuneError {
			return -1
		}

		return r
	}

	return strings.Map(callable, str)
}

func (imap *imap) fetchMessages(seqset *i.SeqSet, mailsChan chan *mail) error {
	messages := make(chan *i.Message)
	section := new(i.BodySectionName)

	go func() {
		if err := imap.Client.UidFetch(seqset, []i.FetchItem{section.FetchItem(), i.FetchEnvelope}, messages); err != nil {
			log.Println(err)
		}
	}()

	for message := range messages {
		mail := new(mail)
		mail.fetchMeta(message)

		reader := message.GetBody(section)

		if reader == nil {
			return errors.New("no reader")
		}

		mailReader, err := m.CreateReader(reader)

		if err != nil {
			mail.Error = err
			mailsChan <- mail

			if mailReader != nil {
				if err := mailReader.Close(); err != nil {
					log.Fatal(err)
				}
			}

			continue
		}

		mail.Error = mail.fetchBody(mailReader)
		mailsChan <- mail

		if mailReader != nil {
			if err := mailReader.Close(); err != nil {
				log.Fatal(err)
			}
		}
	}

	return nil
}
