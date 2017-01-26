// +build darwin
package main

import (
	//"github.com/deckarep/gosx-notifier"
	"github.com/Sirupsen/logrus"
	"github.com/twstrike/gosx-notifier"
)

func NotifyDesktop() {
	note := gosxnotifier.NewNotification("Console succesfully mounted")
	note.Title = "CodePicnic"
	//note.AppIcon = share_dir_darwin + "/" + notify_file
	//note.ContentImage = getHomeDir() + "/" + cfg_dir + "/" + notify_file
	//note.Sender = "com.apple.Safari"

	err := note.Push()
	if err != nil {
		logrus.Errorf("Can't notify %v", err)
	}

}
