import os
import json
path = 'item'
config = {}
for fn in os.listdir(path):
    if os.path.isfile(os.path.join(path, fn)):
        if fn[-5:] == '.json':
            #print(fn)
            with open(os.path.join(path, fn), 'r') as fp:
                obj = json.load(fp)
            obj['asset'] = []
            config[fn[:-5]] = obj
print(json.dumps(config, indent=4))
#with open(os.path.join(path, fn), 'w') as fp:
#    json.dump(obj, fp, indent=4)
                
