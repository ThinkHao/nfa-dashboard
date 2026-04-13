package service

import (
	"crypto/rand"
	"fmt"
	"strings"
	"unicode"

	"nfa-dashboard/internal/model"

	pinyin "github.com/mozillazg/go-pinyin"
	"golang.org/x/crypto/bcrypt"
)

type CustomerRateOwnerIDs struct {
	CustomerFeeOwnerID    *uint64
	NetworkLineFeeOwnerID *uint64
	GeneralFeeOwnerID     *uint64
	ChannelOwnerUserID    *uint64
}

type MissingCustomerRateOwner struct {
	Field string `json:"field"`
	Alias string `json:"alias"`
}

type MissingImportUser struct {
	Alias             string   `json:"alias"`
	SuggestedUsername string   `json:"suggested_username"`
	Fields            []string `json:"fields,omitempty"`
	Lines             []int    `json:"lines,omitempty"`
}

type CreatedImportUser struct {
	Alias    string `json:"alias"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *ratesService) LookupCustomerRateOwnerIDsByDisplayName(names CustomerRateOwnerNames) (CustomerRateOwnerIDs, []MissingCustomerRateOwner, error) {
	if s.userRepo == nil {
		return CustomerRateOwnerIDs{}, nil, NewBadRequest("user repository is required")
	}
	type ownerNameRef struct {
		field string
		value string
		set   func(*CustomerRateOwnerIDs, uint64)
	}
	refs := []ownerNameRef{
		{
			field: "客户费归属",
			value: strings.TrimSpace(names.CustomerFeeOwnerName),
			set:   func(ids *CustomerRateOwnerIDs, id uint64) { ids.CustomerFeeOwnerID = &id },
		},
		{
			field: "线路费归属",
			value: strings.TrimSpace(names.NetworkLineFeeOwnerName),
			set:   func(ids *CustomerRateOwnerIDs, id uint64) { ids.NetworkLineFeeOwnerID = &id },
		},
		{
			field: "节点通用费归属",
			value: strings.TrimSpace(names.GeneralFeeOwnerName),
			set:   func(ids *CustomerRateOwnerIDs, id uint64) { ids.GeneralFeeOwnerID = &id },
		},
		{
			field: "渠道费归属",
			value: strings.TrimSpace(names.ChannelOwnerName),
			set:   func(ids *CustomerRateOwnerIDs, id uint64) { ids.ChannelOwnerUserID = &id },
		},
	}

	lookupNames := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.value != "" {
			lookupNames = append(lookupNames, ref.value)
		}
	}
	if len(lookupNames) == 0 {
		return CustomerRateOwnerIDs{}, nil, nil
	}

	users, err := s.userRepo.FindActiveByDisplayNames(lookupNames)
	if err != nil {
		return CustomerRateOwnerIDs{}, nil, err
	}

	byDisplayName := make(map[string][]uint64, len(users))
	for _, u := range users {
		name := displayImportUserName(u)
		if name == "" {
			continue
		}
		byDisplayName[name] = append(byDisplayName[name], u.ID)
	}

	resolved := CustomerRateOwnerIDs{}
	missing := make([]MissingCustomerRateOwner, 0)
	for _, ref := range refs {
		if ref.value == "" {
			continue
		}
		matches := byDisplayName[ref.value]
		switch len(matches) {
		case 0:
			missing = append(missing, MissingCustomerRateOwner{Field: ref.field, Alias: ref.value})
		case 1:
			ref.set(&resolved, matches[0])
		default:
			return CustomerRateOwnerIDs{}, nil, NewBadRequest(ref.field + " 存在多个同名用户：" + ref.value)
		}
	}
	return resolved, missing, nil
}

func (s *ratesService) ResolveCustomerRateOwnerIDsByDisplayName(rate *model.RateCustomer, names CustomerRateOwnerNames) error {
	if rate == nil {
		return NewBadRequest("rate is required")
	}
	ids, missing, err := s.LookupCustomerRateOwnerIDsByDisplayName(names)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		first := missing[0]
		return NewBadRequest(first.Field + " 未匹配到系统用户：" + first.Alias)
	}
	if ids.CustomerFeeOwnerID != nil {
		rate.CustomerFeeOwnerID = ids.CustomerFeeOwnerID
	}
	if ids.NetworkLineFeeOwnerID != nil {
		rate.NetworkLineFeeOwnerID = ids.NetworkLineFeeOwnerID
	}
	if ids.GeneralFeeOwnerID != nil {
		rate.GeneralFeeOwnerID = ids.GeneralFeeOwnerID
	}
	if ids.ChannelOwnerUserID != nil {
		rate.ChannelOwnerUserID = ids.ChannelOwnerUserID
	}
	return nil
}

func (s *ratesService) PreviewCustomerRateImportUsers(aliases []string) ([]MissingImportUser, error) {
	uniqAliases := uniqueImportAliases(aliases)
	items := make([]MissingImportUser, 0, len(uniqAliases))
	reserved := make(map[string]struct{}, len(uniqAliases))
	for _, alias := range uniqAliases {
		username, err := s.nextAvailableImportUsername(alias, reserved)
		if err != nil {
			return nil, err
		}
		items = append(items, MissingImportUser{Alias: alias, SuggestedUsername: username})
		reserved[username] = struct{}{}
	}
	return items, nil
}

func (s *ratesService) CreateCustomerRateImportUsers(missing []MissingImportUser) ([]CreatedImportUser, error) {
	if s.userRepo == nil {
		return nil, NewBadRequest("user repository is required")
	}
	reserved := make(map[string]struct{}, len(missing))
	created := make([]CreatedImportUser, 0, len(missing))
	for _, item := range missing {
		alias := strings.TrimSpace(item.Alias)
		if alias == "" {
			continue
		}
		username, err := s.nextAvailableImportUsername(alias, reserved)
		if err != nil {
			return nil, err
		}
		password, err := generateImportUserPassword(12)
		if err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		status := int8(1)
		aliasCopy := alias
		createdUser, err := s.userRepo.Create(&model.User{
			Username:     username,
			Alias:        &aliasCopy,
			PasswordHash: string(hash),
			Status:       status,
		})
		if err != nil {
			if isDuplicateUsernameError(err) {
				for i := 2; i < 1000; i++ {
					candidate := fmt.Sprintf("%s%d", normalizeImportUsernameBase(alias), i)
					candidate = clampImportUsername(candidate)
					if _, ok := reserved[candidate]; ok || candidate == "" {
						continue
					}
					exists, existsErr := s.userRepo.UsernameExists(candidate)
					if existsErr != nil {
						return nil, existsErr
					}
					if exists {
						continue
					}
					createdUser, err = s.userRepo.Create(&model.User{
						Username:     candidate,
						Alias:        &aliasCopy,
						PasswordHash: string(hash),
						Status:       status,
					})
					if err == nil {
						username = candidate
						break
					}
					if !isDuplicateUsernameError(err) {
						return nil, err
					}
				}
			}
			if err != nil {
				return nil, err
			}
		}
		reserved[createdUser.Username] = struct{}{}
		created = append(created, CreatedImportUser{
			Alias:    alias,
			Username: createdUser.Username,
			Password: password,
		})
	}
	return created, nil
}

func (s *ratesService) nextAvailableImportUsername(alias string, reserved map[string]struct{}) (string, error) {
	base := normalizeImportUsernameBase(alias)
	for i := 1; i < 1000; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s%d", base, i)
		}
		candidate = clampImportUsername(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := reserved[candidate]; ok {
			continue
		}
		exists, err := s.userRepo.UsernameExists(candidate)
		if err != nil {
			return "", err
		}
		if exists {
			continue
		}
		return candidate, nil
	}
	return "", NewBadRequest("无法生成可用用户名")
}

func normalizeImportUsernameBase(alias string) string {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "user"
	}
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	parts := pinyin.LazyPinyin(alias, args)
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = sanitizeImportUsernameToken(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return "user"
	}
	var b strings.Builder
	b.WriteString(cleaned[0])
	for _, part := range cleaned[1:] {
		b.WriteByte(part[0])
	}
	result := sanitizeImportUsernameToken(b.String())
	if result == "" {
		return "user"
	}
	return clampImportUsername(result)
}

func sanitizeImportUsernameToken(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func clampImportUsername(username string) string {
	username = sanitizeImportUsernameToken(username)
	if username == "" {
		return "user"
	}
	if len(username) > 64 {
		return username[:64]
	}
	return username
}

func uniqueImportAliases(aliases []string) []string {
	out := make([]string, 0, len(aliases))
	seen := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		out = append(out, alias)
	}
	return out
}

func displayImportUserName(u model.User) string {
	if u.Alias != nil && strings.TrimSpace(*u.Alias) != "" {
		return strings.TrimSpace(*u.Alias)
	}
	return strings.TrimSpace(u.Username)
}

func generateImportUserPassword(length int) (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	if length <= 0 {
		length = 12
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i, b := range buf {
		out[i] = chars[int(b)%len(chars)]
	}
	return string(out), nil
}

func isDuplicateUsernameError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate") || strings.Contains(s, "duplic")
}
