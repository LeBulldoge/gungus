# gungus

A discord bot made for the miscellaneous needs of mine and my friends' server. Intended to be self-hosted.
Docker image is provided at `ghcr.io/lebulldoge/gungus:beta`

### Usage:
```sh
$ gungus -token <your discord app token>
```
### Docker:
```sh
$ docker run -v <path to storage directory>:/config ghcr.io/lebulldoge/gungus -token <your discord app token>
```
### Current functionality:
* User polling

Command `/poll start` generates a poll with up to 6 options, 2 of which are required:

`/poll start title: Favorite Chip option_0: 🔥;Sweet Chili Heat Doritos option_1: 🧀;Chili Cheese Fritos option_2: 🧂;Salt & Vinegar Pringles`
![starting poll](https://github.com/LeBulldoge/gungus/assets/13983982/1cc215a4-b501-4746-9fd9-70deb1583d0b)
