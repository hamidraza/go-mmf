package main

import (
	"log"
	"os"

	i "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	m "github.com/emersion/go-message/mail"
	dotenv "github.com/joho/godotenv"
)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getEnv(key string) string {
	return os.Getenv(key)
}

func main() {
	must(dotenv.Load(".env", ".env.defaults"))

	log.Println("Connecting to server...")
	c, err := client.DialTLS(getEnv("IMAP_HOST"), nil)
	must(err)
	log.Println("Connected")

	defer c.Logout()

	must(c.Login(getEnv("EMAIL_USERNAME"), getEnv("EMAIL_PASSWORD")))
	log.Println("Logged in")

	_, err = c.Select("INBOX", false)
	must(err)

	searchCriteria := i.NewSearchCriteria()
	searchCriteria.Text = []string{getEnv("EMAIL_SEARCH_TEXT")}
	// searchCriteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(searchCriteria)
	must(err)
	// log.Println("Total Email:", len(ids))

	seqset := new(i.SeqSet)
	seqset.AddNum(ids...)

	log.Println("seqset: ", seqset)

	messages := make(chan *i.Message, 1)
	done := make(chan error, 1)
	var section i.BodySectionName

	go func() {
		done <- c.Fetch(seqset, []i.FetchItem{section.FetchItem()}, messages)
	}()

	for msg := range messages {
		mail := new(mail)
		sec := new(i.BodySectionName)
		reader := msg.GetBody(sec)
		mailReader, err := m.CreateReader(reader)
		must(err)

		must(mail.fetchBody(mailReader))
		for _, attachment := range mail.Attachments {
			f, err := os.Create("./" + attachment.Filename)
			must(err)
			_, err1 := f.Write(attachment.Body)
			must(err1)
			log.Println("Saved: " + attachment.Filename)
			f.Close()
		}
		mailReader.Close()

	}

	log.Println("Done!")
}
