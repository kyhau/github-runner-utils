import base64


def read_file(filename):
    with open(filename) as f:
        return f.readlines()


pem_text = "".join(read_file("testdata/sample_key.pem"))
print("CheckPt 1")
print(pem_text)

# Encode pem
message = pem_text
message_bytes = message.encode('utf-8')
base64_bytes = base64.b64encode(message_bytes)
base64string = base64_bytes.decode('utf-8')

print("CheckPt 2")
print(base64string)

# Decodee to get the pem again

pem = base64.b64decode(base64string)
pem = pem.decode("utf-8")

print("CheckPt 3")
print(pem)
