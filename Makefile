build: main.go
	go build -o ./dist/tatami .

run: build
	echo "exec ./dist/tatami" > xinitrc
	./run.fish

clean:
	rm -rf ./dist xinitrc
