package dns

import (
	"errors"
	"domainnest/internal/aliyun"
)

// AliyunProvider wraps the existing aliyun.Client to implement the Provider interface.
type AliyunProvider struct {
	client *aliyun.Client
}

func init() {
	Register("aliyun", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		client, err := aliyun.NewClientFromKeys(accessKeyID, accessKeySecret, endpoint)
		if err != nil {
			return nil, err
		}
		return &AliyunProvider{client: client}, nil
	})
}

func (p *AliyunProvider) GetType() string { return "aliyun" }

func (p *AliyunProvider) ListDomains() ([]Domain, error) {
	aliyunDomains, err := p.client.DescribeDomains()
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(aliyunDomains))
	for i, d := range aliyunDomains {
		domains[i] = Domain{DomainName: d.DomainName, RecordCount: d.RecordCount}
	}
	return domains, nil
}

func (p *AliyunProvider) ListRecords(domainName string) ([]Record, error) {
	aliyunRecords, err := p.client.ListAllRecords(domainName)
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(aliyunRecords))
	for _, r := range aliyunRecords {
		rec := Record{
			RecordID: r.RecordID,
			Host:     r.RR,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      r.TTL,
			Priority: r.Priority,
			Line:     r.Line,
			Enabled:  r.Status == "Enable",
		}
		records = append(records, rec)
	}
	return records, nil
}

func (p *AliyunProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	return p.client.AddRecord(domainName, rr, recordType, value, ttl, priority)
}

func (p *AliyunProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	err := p.client.UpdateRecord(domainName, recordID, rr, recordType, value, ttl, priority)
	if err != nil {
		var dupErr *aliyun.DuplicateRecordError
		if errors.As(err, &dupErr) {
			return &DuplicateRecordError{RecordID: dupErr.RecordID}
		}
	}
	return err
}

func (p *AliyunProvider) DeleteRecord(recordID string) error {
	return p.client.DeleteRecord(recordID)
}
