import os
import json
path = 'script'
for fn in os.listdir(path):
    if os.path.isfile(os.path.join(path, fn)):
        if fn[-5:] == '.json':
            print(fn)
            with open(os.path.join(path, fn), 'r') as fp:
                obj = json.load(fp)
            if 'forceUI' in obj:
                forceUI = obj['forceUI']
                print(forceUI)
                if 'currentItem' in forceUI:
                    forceUI['itemView'] = True
            print(obj)
            with open(os.path.join(path, fn), 'w') as fp:
                json.dump(obj, fp, indent=4)
                
