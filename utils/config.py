# -*- coding: utf-8 -*-
import json
import os


def loadJSON(name):
    path = f"{name}.json"
    if os.path.exists(path):
        with open(path, encoding='utf-8') as file:
            return json.load(file)


def saveJSON(name, dct):
    path = f"{name}.json"
    with open(path, "w", encoding='utf-8') as file:
        json.dump(dct, file, ensure_ascii=False, indent="\t")
