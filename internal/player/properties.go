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

func (p *Player) getStringProperty(property string) (string, error) {

	v, err := p.GetProperty(property)
	if err != nil {
		return "", err
	}

	if v == nil {
		return "", nil
	}

	switch val := v.(type) {

	case string:
		return val, nil

	case map[string]interface{}:
		if t, ok := val["title"].(string); ok {
			return t, nil
		}
	}

	return "", nil
}

func (p *Player) getFloatProperty(property string) (float64, error) {

	v, err := p.GetProperty(property)
	if err != nil {
		return 0, err
	}

	if v == nil {
		return 0, nil
	}

	switch val := v.(type) {

	case float64:
		return val, nil

	case int:
		return float64(val), nil
	}

	return 0, nil
}

func (p *Player) getBoolProperty(property string) (bool, error) {

	v, err := p.GetProperty(property)
	if err != nil {
		return false, err
	}

	if v == nil {
		return false, nil
	}

	if val, ok := v.(bool); ok {
		return val, nil
	}

	return false, nil
}

func (p *Player) GetCurrentTime() (float64, error) {
	return p.getFloatProperty("time-pos")
}

func (p *Player) GetDuration() (float64, error) {
	return p.getFloatProperty("duration")
}

func (p *Player) IsPaused() (bool, error) {
	return p.getBoolProperty("pause")
}

func (p *Player) GetTitle() (string, error) {
	return p.getStringProperty("media-title")
}
