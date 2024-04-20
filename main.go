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
	"time"
)

func forgeSearch(artist string) string {
	return fmt.Sprintf(`https://api.discogs.com/database/search?q=%s&type=master&format=album&artist=%s&per_page=100&token=tgRatMaOmFfXjBwHNBlZDQtXrOAELZwpywEOCEbb`, artist, artist)
}

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

func writeRelease(masterUrls []services.Record) {
	services.FilterReleases(services.ProcessMasterURLs(masterUrls, 2))
}

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
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
	choices []services.Record
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

const (
	InputArtist State = iota
	Searching
	SelectArtist
	Fetching
	SelectReleases
	Done
)

func initialModel() *model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	ti.SetValue("Sonic Youth")
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	li := list.New(nil, list.NewDefaultDelegate(), 0, 0)

	return &model{
		artist:  ti,
		err:     nil,
		records: nil,
		state:   InputArtist,
		list:    li,
		spinner: s,
	}
}

func (m *model) Init() tea.Cmd {
	textinput.Blink()
	return m.spinner.Tick
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.state == Searching|Fetching {
		return m, nil
	}
	updateList := func(m *model) tea.Cmd {
		return func() tea.Msg {
			m.records = getRecords(forgeSearch(m.artist.Value()))
			items := make([]list.Item, len(m.records))
			for i, record := range m.records {
				items[i] = record
			}
			m.list.SetItems(items)
			m.list.Title = "Press Enter to select releases, Space to confirm selection"
			m.state = SelectReleases
			_, cmd := m.list.Update(msg)
			return cmd
		}
	}
	fetchReleases := func(m *model) tea.Cmd {
		return func() tea.Msg {
			writeRelease(m.choices)
			m.state = Done
			_, cmd := m.spinner.Update(spinner.TickMsg{
				Time: time.Now(),
			})
			return cmd
		}
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			switch m.state {
			case InputArtist:
				m.state = Searching
				return m, tea.Batch(m.spinner.Tick, updateList(m))
			case SelectReleases:
				item := m.list.Items()[m.list.Cursor()]
				m.choices = append(m.choices, item.(services.Record))
				m.list.RemoveItem(m.list.Cursor())
				return m, nil
			default:
				return m, nil
			}
		case tea.KeySpace:
			switch m.state {
			case InputArtist:
				m.artist, cmd = m.artist.Update(msg)
				return m, cmd
			case SelectReleases:
				m.state = Fetching
				m.spinner, cmd = m.spinner.Update(msg)
				return m, tea.Batch(cmd, fetchReleases(m))
			default:
				return m, nil
			}

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
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	default:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *model) View() string {
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
	case Done:
		return fmt.Sprintf("\n\n   All done!\n\n%s", "   (ctrl+c to quit)")
	}

}
