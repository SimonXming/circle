package handler

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"github.com/gorilla/securecookie"

	"context"
	"github.com/SimonXming/circle/model"
	"github.com/SimonXming/circle/remote"
	"github.com/SimonXming/circle/store"
	"github.com/SimonXming/circle/utils/httputil"
	"github.com/SimonXming/circle/utils/token"
	"github.com/labstack/echo"
	"net/http"
	"strconv"
)

func NewEchoServer() *echo.Echo {
	return echo.New()
}

func GetRoot(c echo.Context) error {
	info := Config.Services.Queue.Info(context.Background())
	println(len(info.Pending))
	println(len(info.Running))
	fmt.Printf("%v\r\n", info.Stats)
	return c.String(http.StatusOK, "Hello, World!")
}

func GetQueueInfo(c echo.Context) error {
	info := Config.Services.Queue.Info(context.Background())
	return c.JSON(http.StatusOK, info.Stats)
}

func PostScmAccount(c echo.Context) error {
	in := new(model.ScmAccount)
	if err := c.Bind(in); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	account := &model.ScmAccount{
		Host:         in.Host,
		Login:        in.Login,
		Password:     in.Password,
		Type:         in.Type,
		PrivateToken: in.PrivateToken,
	}

	err := store.ScmAccountCreate(c, account)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}
	return nil

}

func GetScmAccounts(c echo.Context) error {
	accounts, err := store.ScmAccountList(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}
	return c.JSON(http.StatusOK, accounts)

}

func GetScmAccount(c echo.Context) error {
	scmId, err := strconv.ParseInt(c.Param("scmID"), 10, 64)
	if err != nil {
		c.Error(err)
		return err
	}
	account, err := store.ScmAccountLoad(c, scmId)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}
	return c.JSON(http.StatusOK, account)

}

func GetRemoteRepos(c echo.Context) error {
	scmId, err := strconv.ParseInt(c.Param("scmID"), 10, 64)
	if err != nil {
		c.Error(err)
		return err
	}
	_, err = store.SetupRemoteWithScmID(c, scmId)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	repos, err := remote.Repos(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, repos)
}

func PostRepo(c echo.Context) error {
	scmId, err := strconv.ParseInt(c.Param("scmID"), 10, 64)
	if err != nil {
		c.Error(err)
		return err
	}

	in := new(model.Repo)
	if err := c.Bind(in); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	_, err = store.SetupRemoteWithScmID(c, scmId)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	r, err := remote.Repo(c, in.Owner, in.Name)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	// error if the repository already exists
	_, err = store.GetRepoScmIDOwnerName(c, scmId, r.Owner, r.Name)
	if err == nil {
		c.String(http.StatusConflict, "Repository already exists.")
		return err
	}
	r.ScmId = scmId
	r.AllowPush = true
	r.AllowPull = true
	r.Hash = base32.StdEncoding.EncodeToString(
		securecookie.GenerateRandomKey(32),
	)

	// crates the jwt token used to verify the repository
	t := token.New(token.HookToken, r.FullName)
	sig, err := t.Sign(r.Hash)
	if err != nil {
		c.String(http.StatusBadRequest, "Generate webhook token failed.")
		return err
	}

	link := fmt.Sprintf(
		"%s/hook?access_token=%s",
		httputil.GetURL(c.Request()),
		sig,
	)

	// activate the repository before we make any
	// local changes to the database.
	err = remote.Activate(c, r, link)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	err = store.RepoCreate(c, r)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}
	return nil
}

func PostConfig(c echo.Context) error {
	repoID, err := strconv.ParseInt(c.Param("repoID"), 10, 64)
	if err != nil {
		c.Error(err)
		return err
	}

	repo, err := store.RepoLoad(c, repoID)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	in := new(model.Config)
	if err := c.Bind(in); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	config := &model.Config{
		RepoID: repo.ID,
		Data:   in.Data,
	}
	config.Hash = shasum([]byte(in.Data))

	err = store.ConfigCreate(c, config)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}
	return nil
}

func PostSecret(c echo.Context) error {
	in := new(model.Secret)
	if err := c.Bind(in); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}

	secret := &model.Secret{
		RepoID: 123,
		Name:   in.Name,
		Value:  in.Value,
	}

	err := store.SecretCreate(c, secret)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}
	return nil
}

func shasum(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum)
}
