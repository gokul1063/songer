package player

import (
	"encoding/json"
)

func (p *Player) GetProperty(property string) (interface{}, error) {

	p.mu.Lock()
	defer p.mu.Unlock()

	req := map[string]interface{}{
		"command": []interface{}{"get_property", property},
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	data = append(data, '\n')

	_, err = p.conn.Write(data)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 4096)

	n, err := p.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	var resp map[string]interface{}

	err = json.Unmarshal(buf[:n], &resp)
	if err != nil {
		return nil, err
	}

	return resp["data"], nil
}

func (p *Player) GetCurrentTime() (float64, error) {

	v, err := p.GetProperty("time-pos")
	if err != nil {
		return 0, err
	}

	if v == nil {
		return 0, nil
	}

	return v.(float64), nil
}

func (p *Player) GetDuration() (float64, error) {

	v, err := p.GetProperty("duration")
	if err != nil {
		return 0, err
	}

	if v == nil {
		return 0, nil
	}

	return v.(float64), nil
}

func (p *Player) IsPaused() (bool, error) {

	v, err := p.GetProperty("pause")
	if err != nil {
		return false, err
	}

	if v == nil {
		return false, nil
	}

	return v.(bool), nil
}

func (p *Player) GetTitle() (string, error) {

	v, err := p.GetProperty("media-title")
	if err != nil {
		return "", err
	}

	if v == nil {
		return "", nil
	}

	return v.(string), nil
}
