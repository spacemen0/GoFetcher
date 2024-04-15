package main

import (
	"GoFetcher/services"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"log"
	"net/http"
)

const url = "https://api.discogs.com/artists/3840/releases"

func getRecords(url string) []services.Record {
	resp, err := services.SendRequest(url)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error: Unexpected status code:", resp.StatusCode)
		return nil
	}

	data, err := services.DecodeJSON(resp)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	return services.FilterMasterURLs(data)
}

func extractUrl(records []services.Record) []any {
	var masterUrls []any
	for _, record := range records {
		masterUrls = append(masterUrls, record.Description())
	}
	return masterUrls
}

func writeRelease(masterUrls []any) {
	services.WriteToFile(services.FilterReleases(services.ProcessMasterURLs(masterUrls)))
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type (
	errMsg error
)

type State int8

type model struct {
	artist  textinput.Model
	err     error
	records []services.Record
	state   State
	list    list.Model
	spinner spinner.Model
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

const (
	InputArtist State = iota
	Searching
	SelectArtist
	Fetching
	SelectReleases
	WriteFile
)

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Sonic Youth"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	li := list.New(nil, list.NewDefaultDelegate(), 0, 0)

	return model{
		artist:  ti,
		err:     nil,
		records: nil,
		state:   InputArtist,
		list:    li,
		spinner: s,
	}
}

func (m model) Init() tea.Cmd {
	textinput.Blink()
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			m.state = Searching

			m.records = getRecords(url)
			items := make([]list.Item, len(m.records))
			for i, record := range m.records {
				items[i] = record
			}
			m.list.SetItems(items)
			m.state = SelectReleases
			return m, nil
		default:
			if m.state == InputArtist {
				m.artist, cmd = m.artist.Update(msg)
				return m, cmd
			}
			if m.state == SelectReleases {
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			}
			return m, nil

		}
	case errMsg:
		m.err = msg
		return m, nil
	default:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	switch m.state {
	default:
		return fmt.Sprintf(
			"Type the name of the artist you want to search\n\n%s\n\n%s",
			m.artist.View(),
			"(esc to quit)",
		) + "\n"
	case InputArtist:
		return fmt.Sprintf(
			"Type the name of the artist you want to search\n\n%s\n\n%s",
			m.artist.View(),
			"(esc to quit)",
		) + "\n"
	case Searching:
		return fmt.Sprintf("\n\n   %s Searching...\n\n", m.spinner.View())
	case SelectArtist:
		return docStyle.Render(m.list.View())
	case Fetching:
		return fmt.Sprintf("\n\n   %s Fetching Releases...\n\n", m.spinner.View())
	case SelectReleases:
		return docStyle.Render(m.list.View())
	case WriteFile:
		return fmt.Sprintf("\n\n   All done!\n\n")
	}

}
