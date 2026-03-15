build: main.go
	go build -o ./dist/tatami .

run: build
	printf "#!/bin/sh\nexport DISPLAY=:100\nexec ./dist/tatami -mod mod1 -launcher dmenu_run" > xinitrc
	./run.fish

clean:
	rm -rf ./dist xinitrc
