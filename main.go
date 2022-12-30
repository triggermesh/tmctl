/*
Copyright 2022 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/triggermesh/tmctl/cmd"
	"github.com/triggermesh/tmctl/cmd/brokers"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
	"github.com/triggermesh/tmctl/test"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

var (
	Version string = "dev"
	Commit  string = "unknown"
)

// func mthd() {
// 	if err := cmd.NewRootCommand(Version, Commit).Execute(); err != nil {
// 		log.Fatal(err)
// 	}
// }

func main() {
	if len(os.Args) > 1 {
		if err := cmd.NewRootCommand(Version, Commit).Execute(); err != nil {
			log.Fatal(err)
		}

	} else {
		gui()
	}
}

func gui() {
	myApp := app.New()
	myWindow := myApp.NewWindow("tmctl")
	myWindow.Resize(fyne.NewSize(1080, 1080))

	// create tabs for the following:
	// 1. brokers
	// 2. config
	// 3. create
	// 4. delete
	// 5. describe
	// 6. dump
	// 7. sendevent
	// 8. start
	// 9. stop
	// 10. version
	// 11. watch

	userInputBrokerNameBinding := binding.NewString()
	userInputInput := widget.NewForm(
		widget.NewFormItem("Broker Name:", widget.NewEntryWithData(userInputBrokerNameBinding)),
		// add a sanitization function to the entry
	)

	brokers, err := listBrokers()
	if err != nil {
		fmt.Println(err)
	}

	brokerDataBinding := binding.BindStringList(&brokers)

	//create a list of brokers with an attached "delete" button
	brokerList := widget.NewListWithData(brokerDataBinding, func() fyne.CanvasObject {
		return container.NewHBox(
			widget.NewLabel("Avalible Brokers:"),
		)
	}, func(item binding.DataItem, itemObject fyne.CanvasObject) {
		itemObject.(*fyne.Container).Objects[0].(*widget.Label).Bind(item.(binding.String))
		itemObject.(*fyne.Container).Add(widget.NewButton("Delete", func() {
			fmt.Println("Delete")
			br, err := item.(binding.String).Get()
			if err != nil {
				fmt.Println(err)
			}
			deleteBroker(br)
		}))
	})

	brokerRefreshButton := widget.NewButton("Refresh", func() {
		brokers, err = listBrokers()
		if err != nil {
			fmt.Println(err)
		}

		if err := brokerDataBinding.Set(brokers); err != nil {
			fmt.Println(err)
		}
		brokerList.Refresh()
	})

	brokerTab := container.NewVBox(
		userInputInput,
		widget.NewButton("Create", func() {
			ui, err := userInputBrokerNameBinding.Get()
			if err != nil {
				fmt.Println(err)
			}
			createBroker(ui)
		}),
		brokerRefreshButton,
		brokerList,
	)

	configTab := container.NewVBox(
		widget.NewLabel("Config"),
		widget.NewButton("List", func() {
			fmt.Println("List")
		}),
	)

	avalibleSources, err := crd.ListSources(test.CRD())
	if err != nil {
		fmt.Println(err)
	}

	acordianWidget := &widget.Accordion{}
	for _, source := range avalibleSources {
		acitem := container.NewVBox(
			widget.NewLabel(source),
			widget.NewButton("Create", func() {
				fmt.Println("Create")
			}))
		acordianWidget.Append(widget.NewAccordionItem(source, acitem))
	}

	// // call the crd.Process function to get a list of information on the avalible sources
	// found, properties := completion.SpecFromCRD("AWSS3Source", test.CRD(), "destination")
	// // found, properties := completion.SpecFromCRD("AWSS3Source", test.CRD(), "auth", "credentials")
	// if !found {
	// 	fmt.Println("not found")
	// }

	// fmt.Printf("%+v", properties)
	// for _, property := range properties {
	// 	fmt.Printf("%+v", property)
	// }

	// var props = map[string]crd.Property{}

	createTab := container.NewVBox(
		widget.NewLabel("Create"),
		acordianWidget,
	)

	deleteTab := container.NewVBox(
		widget.NewLabel("Delete"),
		widget.NewButton("List", func() {
			fmt.Println("List")
		}),
	)

	describeTab := container.NewVBox(
		widget.NewLabel("Describe"),
		widget.NewButton("List", func() {
			fmt.Println("List")
		}),
	)

	var dumpData []string
	dumpDataBinding := binding.BindStringList(&dumpData)

	dumpList := widget.NewListWithData(dumpDataBinding, func() fyne.CanvasObject {
		return container.NewHBox(
			widget.NewLabel("Dump Data:"),
		)
	}, func(item binding.DataItem, itemObject fyne.CanvasObject) {
		itemObject.(*fyne.Container).Objects[0].(*widget.Label).Bind(item.(binding.String))
	})

	dumpTab := container.NewVBox(
		widget.NewLabel("Dump"),
		dumpList,
		widget.NewButton("Dump", func() {
			fmt.Println("Dump")
			dumpData, err = dump()
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println("displaying dump data")
			for _, data := range dumpData {
				fmt.Println(data)
			}

			if err := dumpDataBinding.Set(dumpData); err != nil {
				fmt.Println(err)
			}
			dumpList.Refresh()

		}),
	)

	// SEND EVENTS START HERE

	uiEventTypeBinding := binding.NewString()
	uiEventTypeWidget := widget.NewForm(
		widget.NewFormItem("Event Type:", widget.NewEntryWithData(uiEventTypeBinding)),
	)

	uiEventTargetBinding := binding.NewString()
	uiEventTargetWidget := widget.NewForm(
		widget.NewFormItem("Event Target:", widget.NewEntryWithData(uiEventTargetBinding)),
	)

	uiEventPayloadBinding := binding.NewString()
	uiEventPayloadWidget := widget.NewForm(
		widget.NewFormItem("Event Payload:", widget.NewEntryWithData(uiEventPayloadBinding)),
	)

	sendEventTab := container.NewVBox(
		widget.NewLabel("SendEvent"),
		uiEventTypeWidget,
		uiEventTargetWidget,
		uiEventPayloadWidget,
		widget.NewButton("Send", func() {
			eventType, err := uiEventTypeBinding.Get()
			if err != nil {
				fmt.Println(err)
			}

			eventTarget, err := uiEventTargetBinding.Get()
			if err != nil {
				fmt.Println(err)
			}

			eventPayload, err := uiEventPayloadBinding.Get()
			if err != nil {
				fmt.Println(err)
			}

			sendEvent(eventType, eventTarget, eventPayload)
		}),
	)

	// SEND EVENTS END HERE

	startTab := container.NewVBox(
		widget.NewLabel("Start"),
		widget.NewButton("List", func() {
			fmt.Println("List")
		}),
	)

	stopTab := container.NewVBox(
		widget.NewLabel("Stop"),
		widget.NewButton("List", func() {
			fmt.Println("List")
		}),
	)

	versionTab := container.NewVBox(
		widget.NewLabel("Version"),
		widget.NewButton("List", func() {
			fmt.Println("List")
		}),
	)

	watchTab := container.NewVBox(
		widget.NewLabel("Watch"),
		widget.NewButton("List", func() {
			fmt.Println("List")
		}),
	)

	// create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Brokers", brokerTab),
		container.NewTabItem("Config", configTab),
		container.NewTabItem("Create", createTab),
		container.NewTabItem("Delete", deleteTab),
		container.NewTabItem("Describe", describeTab),
		container.NewTabItem("Dump", dumpTab),
		container.NewTabItem("SendEvent", sendEventTab),
		container.NewTabItem("Start", startTab),
		container.NewTabItem("Stop", stopTab),
		container.NewTabItem("Version", versionTab),
		container.NewTabItem("Watch", watchTab),
	)

	myWindow.SetContent(tabs)
	myWindow.ShowAndRun()
}

func createBroker(name string) {
	// check for any " " in the name
	if strings.Contains(name, " ") {
		fmt.Println("Broker name cannot contain spaces")
		return
	}

	// create broker
	fmt.Println("Creating broker: " + name)
	// execute the command to create the broker
	// tmctl create broker foo
	createCmd := exec.Command("./tmctl", "create", "broker", name)
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		log.Fatal(err)
	}

}

func listBrokers() ([]string, error) {
	// // fetch the $USER directory
	// usr, err := user.Current()
	// if err != nil {
	// 	return nil, err
	// }

	return brokers.List("/Users/jeffreynaef/.triggermesh/cli", "")
	// return brokers.List(usr.HomeDir+"/.triggermesh/cli", "")
}

func deleteBroker(name string) {
	// delete broker
	fmt.Println("Deleting broker: " + name)
	// execute the command to delete the broker
	// tmctl delete broker foo
	deleteCmd := exec.Command("./tmctl", "delete", "--broker", name)
	deleteCmd.Stdout = os.Stdout
	deleteCmd.Stderr = os.Stderr
	if err := deleteCmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func sendEvent(eventType, target, data string) error {
	fmt.Println("Sending event: " + eventType + " to " + target + " with data: " + data)
	sendCmd := exec.Command("./tmctl", "send", "event", eventType, "--target", target, data)
	sendCmd.Stdout = os.Stdout
	sendCmd.Stderr = os.Stderr
	if err := sendCmd.Run(); err != nil {
		log.Fatal(err)
	}
	return nil
}

func dump() ([]string, error) {
	dumpCMD := exec.Command("./tmctl", "dump")
	out, err := dumpCMD.Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(out))
	return strings.Split(string(out), "\n"), nil
}
