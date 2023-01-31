import requests
from bs4 import BeautifulSoup
from ast import literal_eval
import time
import logging
logger = logging.getLogger('bot')

def findInRasp(req) -> dict:
	"""Поиск группы (преподавателя) в расписании"""
	logger.debug(f'Find {req}')
	
	rasp = requests.Session() 
	rasp.headers['User-Agent'] = 'Mozilla/5.0'
	hed = rasp.get("https://ssau.ru/rasp/")
	if hed.status_code == 200:
		soup = BeautifulSoup(hed.text, 'lxml')
		csrf_token = soup.select_one('meta[name="csrf-token"]')['content']
	else:
		return 'Error'
	
	time.sleep(1)
	
	rasp.headers['Accept'] = 'application/json'
	rasp.headers['X-CSRF-TOKEN'] = csrf_token
	result = rasp.post("https://ssau.ru/rasp/search", data = {'text':req})
	if result.status_code == 200:
		num = literal_eval(result.text)
	else:
		return 'Error'

	if len(num) == 0:
		return None
	else:
		return num[0]