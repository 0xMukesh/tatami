build: main.go
	go build -o ./dist/tatami .

run: build
	printf "#!/bin/sh\nexport DISPLAY=:100\nexec ./dist/tatami launch" > xinitrc
	./run.fish

clean:
	rm -rf ./dist xinitrc
