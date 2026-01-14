#include <queue>
#include <iostream>

int main()
{
	std::vector<int> arr({3,0,1,2});
	std::priority_queue<int> q(arr.begin(), arr.end());
	while(!q.empty()){
		std::cout << q.top() << std::endl;
		q.pop();
	}
	return 0;
}




