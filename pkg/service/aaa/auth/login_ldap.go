package auth

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"kubegems.io/pkg/log"
)

type LdapLoginUtils struct {
	Name         string `yaml:"name" json:"name"`
	LdapAddr     string `yaml:"addr" json:"ldapaddr"`
	BaseDN       string `yaml:"basedn" json:"basedn"`
	BindUsername string `yaml:"binduser" json:"binduser"`
	BindPassword string `yaml:"bindpass" json:"password"`
}

func (ut *LdapLoginUtils) LoginAddr() string {
	return DefaultLoginURL
}

func (ut *LdapLoginUtils) GetUserInfo(ctx context.Context, cred *Credential) (ret *UserInfo, err error) {
	if !ut.ValidateCredential(cred) {
		return nil, fmt.Errorf("invalid credential")
	}
	ldapConn, err := ldap.Dial("tcp", ut.LdapAddr)
	if err != nil {
		log.Error(err, "connect to ldap server failed")
		return
	}
	defer ldapConn.Close()

	if err = ldapConn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		log.Error(err, "connect to ldap server with tls failed")
		return
	}

	if err = ldapConn.Bind(ut.BindUsername, ut.BindPassword); err != nil {
		log.Error(err, "connect to ldap server with tls failed")
	}

	searchRequest := ldap.NewSearchRequest(
		ut.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(&(objectClass=organizationalPerson)(cn=%s))", cred.Username),
		[]string{"mail"},
		nil,
	)
	result, err := ldapConn.Search(searchRequest)
	if err != nil {
		log.Error(err, "search user in ldap failed", "credential", cred)
		return
	}
	if len(result.Entries) != 1 {
		log.Error(fmt.Errorf("more than one search result returnd"), "credential", cred)
		return
	}
	info := result.Entries[0]
	ret.Username = info.GetAttributeValue("username")
	mailstr := info.GetAttributeValue("mail")
	emailstr := info.GetAttributeValue("email")
	if emailstr != "" {
		ret.Email = emailstr
	} else {
		ret.Email = mailstr
	}
	ret.Source = cred.Source
	return
}

func (ut *LdapLoginUtils) ValidateCredential(cred *Credential) bool {
	userdn := fmt.Sprintf(ut.BaseDN, cred.Username)
	req := ldap.NewSimpleBindRequest(userdn, cred.Password, nil)
	ldapConn, err := ldap.Dial("tcp", ut.LdapAddr)
	if err != nil {
		log.Error(err, "connect to ldap server failed")
		return false
	}
	defer ldapConn.Close()

	if err := ldapConn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		log.Error(err, "connect to ldap server with tls failed")
		return false
	}
	_, err = ldapConn.SimpleBind(req)
	if err != nil {
		log.Info("user login with ldap failed", "credential", cred)
		return false
	}
	return true
}
