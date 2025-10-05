echo "Hello from container!"
echo "Container is working correctly"
echo "Testing streaming output..."
i=1
while [ $i -le 10 ]; do
    echo "Line $i - streaming test"
    sleep 1
    i=$((i + 1))
done
echo "Streaming test complete"
