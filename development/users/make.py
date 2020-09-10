import json
data = {}
with open('usernames.txt', 'r') as fp:
    for line in fp.readlines():
        data[line.strip()] = {
            'progress': [
                {
                    'Scene': '1_gamestart',
                    'Position': 0
                }
            ],
            'itemList': [],
            'UI': {
                'QR': False,
                'itemMenu': False,
                'itemView': False,
                'history': False,
                'currentItem': ''
            }
        }

obj = {'data': data}
print(json.dumps(obj, indent=4))
