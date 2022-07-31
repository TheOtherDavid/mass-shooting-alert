# gun-violence-alert

With the mass shootings plaguing America, it's vital to get the most up-to-date information on the latest bloodbath, so we can offer our thoughts and prayers and get back to doing absolutely nothing to stop it from happening again. To further this aim, I've written an alert system to give a visual indicator whenever a new mass shooting is detected. You will be kept disturbingly aware of the violence that surrounds us all.

This program will get the latest mass shooting data from
massshootingtracker.site
and compare it to the last observed shooting, stored in a config file.
If a new shooting is detected, a command will be sent to a locally running instance of WLED (https://github.com/Aircoookie/WLED) and a red pulse will be displayed for a configurable length of time. Another command will then be sent to tell WLED to return to the previously displayed settings.
This code should be fairly configurable. Right now, it's triggering WLED, but it could equally easily be connected to a Twitter bot to make an announcement, an SMS client to notify you quietly, a speaker to notify you LOUDLY, or even a thermostat, so you can suffer in the summer along with the victims! The possibilities are endless!
